package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	"distributed-kv-raft/kv"
	"distributed-kv-raft/metrics"
	"distributed-kv-raft/raft"
	"distributed-kv-raft/rpc"
)

func main() {
	var (
		nodeID    = flag.String("id", "1", "Node ID")
		httpAddr  = flag.String("http_addr", ":8001", "HTTP address for client API")
		raftAddr  = flag.String("raft_addr", ":9001", "Raft gRPC address")
		peersFlag = flag.String("peers", "", "Comma-separated raft peer addresses")
	)
	flag.Parse()

	peers := []string{}
	if *peersFlag != "" {
		peers = strings.Split(*peersFlag, ",")
	}

	// Initialize metrics
	m := metrics.NewRaftMetrics(*nodeID)

	applyCh := make(chan *rpc.LogEntry)
	rf := raft.NewRaft(*nodeID, peers, applyCh)
	rf.SetMetrics(m)
	
	store := kv.NewStore()
	store.SetMetrics(m)

	// Start state machine applicator
	go func() {
		for entry := range applyCh {
			parts := strings.SplitN(entry.Command, " ", 3)
			if len(parts) < 2 {
				continue
			}
			
			switch parts[0] {
			case "SET":
				if len(parts) >= 3 {
					store.Set(parts[1], parts[2])
					log.Printf("Applied: SET %s = %s", parts[1], parts[2])
				}
			case "DELETE":
				store.Delete(parts[1])
				log.Printf("Applied: DELETE %s", parts[1])
			}
		}
	}()

	// Start metrics updater
	go startMetricsUpdater(m, rf)

	// Start RPC server
	lis, err := net.Listen("tcp", *raftAddr)
	if err != nil {
		log.Fatalf("failed to listen on raft addr %s: %v", *raftAddr, err)
	}
	log.Printf("Raft RPC listening on %s", *raftAddr)

	s := grpc.NewServer()
	rpc.RegisterRaftServiceServer(s, &RaftServer{rf: rf})
	go s.Serve(lis)

	// Expose metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Wrap set handler with CORS
	originalSetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		var key, val string
		// Try parsing JSON body first
		if r.Method == http.MethodPost {
			var body struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				key = body.Key
				val = body.Value
			}
		}
		// Fallback to query params
		if key == "" {
			key = r.URL.Query().Get("key")
		}
		if val == "" {
			val = r.URL.Query().Get("value")
		}

		if key == "" || val == "" {
			http.Error(w, "Missing key/value", 400)
			return
		}

		idx, term, isLeader := rf.StartCommand(fmt.Sprintf("SET %s %s", key, val))
		if !isLeader {
			fmt.Fprintf(w, "Not Leader. Leader is %s", "Unknown") // rf.GetState()
			return
		}
		fmt.Fprintf(w, "Submitted at index %d term %d", idx, term)
	})
	http.Handle("/set", originalSetHandler)

	// Wrap get handler with CORS
	originalGetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		key := r.URL.Query().Get("key")
		val, ok := store.Get(key)
		if !ok {
			http.Error(w, "Not found", 404)
			return
		}
		fmt.Fprintf(w, "%s", val)
	})
	http.Handle("/get", originalGetHandler)

	// ========== Membership Management Endpoints ==========
	
	// GET /peers - List current cluster membership
	membershipInfoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		info := rf.GetMembershipInfo()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})
	http.Handle("/peers", membershipInfoHandler)

	// POST /add-peer?id=node4&addr=10.0.4.10:9000 - Add a new peer to cluster
	addPeerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		nodeID := r.URL.Query().Get("id")
		nodeAddr := r.URL.Query().Get("addr")
		
		if nodeID == "" || nodeAddr == "" {
			http.Error(w, "Missing id or addr parameter", 400)
			return
		}
		
		// Only leader can add peers
		if !rf.IsLeader() {
			http.Error(w, "Only leader can add peers", 500)
			return
		}
		
		// Append membership change to log
		idx, term, err := rf.AppendMembershipChange(nodeID, nodeAddr, "ADD")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to add peer: %v", err), 500)
			return
		}
		
		// Apply immediately (in production, wait for replication)
		err = rf.ApplyMembershipChange(nodeID, nodeAddr, "ADD")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to apply membership change: %v", err), 500)
			return
		}
		
		response := fmt.Sprintf("Peer %s (%s) added at log index %d term %d", nodeID, nodeAddr, idx, term)
		fmt.Fprintf(w, response)
		log.Println(response)
	})
	http.Handle("/add-peer", addPeerHandler)

	// POST /remove-peer?id=node4 - Remove a peer from cluster
	removePeerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		nodeID := r.URL.Query().Get("id")
		
		if nodeID == "" {
			http.Error(w, "Missing id parameter", 400)
			return
		}
		
		// Only leader can remove peers
		if !rf.IsLeader() {
			http.Error(w, "Only leader can remove peers", 500)
			return
		}
		
		// Append membership change to log
		idx, term, err := rf.AppendMembershipChange(nodeID, "", "REMOVE")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to remove peer: %v", err), 500)
			return
		}
		
		// Apply immediately (in production, wait for replication)
		err = rf.ApplyMembershipChange(nodeID, "", "REMOVE")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to apply membership change: %v", err), 500)
			return
		}
		
		response := fmt.Sprintf("Peer %s removed at log index %d term %d", nodeID, idx, term)
		fmt.Fprintf(w, response)
		log.Println(response)
	})
	http.Handle("/remove-peer", removePeerHandler)

	// ========== Advanced API Endpoints (Task 9) ==========

	// GET /range?start=a&end=z - Get all keys in range [start, end)
	rangeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if handleCORSPreflight(w, r) {
			return
		}

		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")
		if start == "" {
			http.Error(w, "Missing 'start' parameter", 400)
			return
		}

		results := store.GetRange(start, end)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"keys":       results,
			"count":      len(results),
			"truncated":  false,
			"range":      map[string]string{"start": start, "end": end},
		})
	})
	http.Handle("/range", rangeHandler)

	// GET /prefix?prefix=user: - Get all keys with prefix
	prefixHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if handleCORSPreflight(w, r) {
			return
		}

		prefix := r.URL.Query().Get("prefix")
		if prefix == "" {
			http.Error(w, "Missing 'prefix' parameter", 400)
			return
		}

		results := store.GetPrefix(prefix)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"keys":      results,
			"count":     len(results),
			"prefix":    prefix,
			"truncated": false,
		})
	})
	http.Handle("/prefix", prefixHandler)

	// GET /keys - List all keys with optional pagination
	keysHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if handleCORSPreflight(w, r) {
			return
		}

		limit := 100
		offset := 0
		if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
			if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
				limit = l
			}
		}
		if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
			if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
				offset = o
			}
		}

		keys, total := store.ListKeysWithLimit(limit, offset)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"keys":       keys,
			"count":      len(keys),
			"total":      total,
			"offset":     offset,
			"limit":      limit,
			"truncated":  offset+limit < total,
			"hasMore":    offset+limit < total,
		})
	})
	http.Handle("/keys", keysHandler)

	// POST /batch/get - Get multiple keys
	batchGetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if handleCORSPreflight(w, r) {
			return
		}

		var body struct {
			Keys []string `json:"keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), 400)
			return
		}

		if len(body.Keys) == 0 {
			http.Error(w, "Empty keys list", 400)
			return
		}

		results := store.GetMultiple(body.Keys)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": results,
			"count":   len(results),
			"found":   len(results),
			"missing": len(body.Keys) - len(results),
		})
	})
	http.Handle("/batch/get", batchGetHandler)

	// POST /batch/set - Set multiple key-value pairs
	batchSetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if handleCORSPreflight(w, r) {
			return
		}

		if !rf.IsLeader() {
			http.Error(w, "Only leader can perform SET operations", 500)
			return
		}

		var body struct {
			Operations []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"operations"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), 400)
			return
		}

		if len(body.Operations) == 0 {
			http.Error(w, "Empty operations list", 400)
			return
		}

		// Submit all operations to Raft
		results := make(map[string]interface{})
		successCount := 0
		for _, op := range body.Operations {
			idx, term, isLeader := rf.StartCommand(fmt.Sprintf("SET %s %s", op.Key, op.Value))
			if isLeader {
				results[op.Key] = map[string]interface{}{
					"success": true,
					"index":   idx,
					"term":    term,
				}
				successCount++
			} else {
				results[op.Key] = map[string]interface{}{
					"success": false,
					"error":   "Not leader",
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": results,
			"total":   len(body.Operations),
			"success": successCount,
			"failed":  len(body.Operations) - successCount,
		})
	})
	http.Handle("/batch/set", batchSetHandler)

	// POST /batch/delete - Delete multiple keys
	batchDeleteHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if handleCORSPreflight(w, r) {
			return
		}

		if !rf.IsLeader() {
			http.Error(w, "Only leader can perform DELETE operations", 500)
			return
		}

		var body struct {
			Keys []string `json:"keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), 400)
			return
		}

		if len(body.Keys) == 0 {
			http.Error(w, "Empty keys list", 400)
			return
		}

		// Submit all deletions to Raft
		results := make(map[string]interface{})
		successCount := 0
		for _, key := range body.Keys {
			idx, term, isLeader := rf.StartCommand(fmt.Sprintf("DELETE %s", key))
			if isLeader {
				results[key] = map[string]interface{}{
					"success": true,
					"index":   idx,
					"term":    term,
				}
				successCount++
			} else {
				results[key] = map[string]interface{}{
					"success": false,
					"error":   "Not leader",
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": results,
			"total":   len(body.Keys),
			"success": successCount,
			"failed":  len(body.Keys) - successCount,
		})
	})
	http.Handle("/batch/delete", batchDeleteHandler)

	// GET /health - Health check endpoint
	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addCORSHeaders(w)
		if handleCORSPreflight(w, r) {
			return
		}

		isLeader := rf.IsLeader()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "healthy",
			"isLeader":       isLeader,
			"role":           map[bool]string{true: "leader", false: "follower"}[isLeader],
			"currentTerm":    rf.GetCurrentTerm(),
			"logSize":        rf.GetLogLength(),
			"commitIndex":    rf.GetCommitIndex(),
			"lastApplied":    rf.GetLastApplied(),
			"storeSize":      store.Count(),
			"timestamp":      time.Now().Unix(),
		})
	})
	http.Handle("/health", healthHandler)


	go func() {
		if err := http.ListenAndServe(*httpAddr, nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal, closing storage...")
		if err := rf.Close(); err != nil {
			log.Printf("Error closing raft storage: %v", err)
		}
		os.Exit(0)
	}()

	// Block main
	select {}
}

// startMetricsUpdater runs background goroutine that periodically updates metrics
func startMetricsUpdater(m *metrics.RaftMetrics, rf *raft.Raft) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Update state metrics
		m.CurrentTerm.Set(float64(rf.GetCurrentTerm()))
		m.LogSize.Set(float64(rf.GetLogLength()))
		m.LastAppliedIndex.Set(float64(rf.GetLastApplied()))
		m.CommitIndex.Set(float64(rf.GetCommitIndex()))
		m.LastIncludedIndex.Set(float64(rf.GetLastIncludedIndex()))

		if rf.IsLeader() {
			m.LeaderID.Set(1)
		} else {
			m.LeaderID.Set(0)
		}
	}
}

// RaftServer adapter
type RaftServer struct {
	rpc.UnimplementedRaftServiceServer
	rf *raft.Raft
}

func (s *RaftServer) RequestVote(ctx context.Context, req *rpc.RequestVoteRequest) (*rpc.RequestVoteResponse, error) {
	return s.rf.HandleRequestVote(req), nil
}

func (s *RaftServer) AppendEntries(ctx context.Context, req *rpc.AppendEntriesRequest) (*rpc.AppendEntriesResponse, error) {
	return s.rf.HandleAppendEntries(req), nil
}

// Helper function to add CORS headers to response
func addCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// Helper function to handle CORS preflight requests
func handleCORSPreflight(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}
