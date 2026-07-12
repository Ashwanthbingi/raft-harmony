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
			if len(parts) >= 3 && parts[0] == "SET" {
				store.Set(parts[1], parts[2])
				log.Printf("Applied: %s = %s", parts[1], parts[2])
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

	// Simple HTTP API for Client
	// Simple HTTP API for Client
	http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
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

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		val, ok := store.Get(key)
		if !ok {
			http.Error(w, "Not found", 404)
			return
		}
		fmt.Fprintf(w, "%s", val)
	})

	log.Printf("Starting HTTP server on %s", *httpAddr)
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
