package raft

import (
	"fmt"
	"log"
	"sync"
	"time"

	"distributed-kv-raft/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Logger interface for structured logging (optional, or just use method)
// We will implement a helper method.

// Peer interface removed as per instructions.

// State represents the current state of a Raft node.
type State int

const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// Raft implements a single node in the Raft cluster.
type Raft struct {
	mu sync.Mutex

	id    string
	peers []string

	// Persistent state on all servers
	currentTerm int32
	votedFor    string
	log         []*rpc.LogEntry

	// Volatile state on all servers
	commitIndex int32
	lastApplied int32

	// Volatile state on leaders
	nextIndex  map[string]int32
	matchIndex map[string]int32

	// State
	state    State
	leaderId string

	// Channels/Timers
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer
	lastHeartbeat  time.Time

	// Apply channel
	applyCh chan *rpc.LogEntry

	// Connection pool
	muConns sync.Mutex
	conns   map[string]*grpc.ClientConn

	// Persistent storage
	storage *PersistentStorage

	// Snapshot metadata (for log compaction)
	lastIncludedIndex int32 // Last log index included in snapshot
	lastIncludedTerm  int32 // Term of last_included_index

	// Membership management for dynamic cluster changes
	membership *MembershipConfig

	// Metrics for observability
	metrics interface{} // *metrics.RaftMetrics (avoid circular import)
}

// NewRaft creates a new Raft node.
func NewRaft(id string, peers []string, applyCh chan *rpc.LogEntry) *Raft {
	// Initialize persistent storage
	dbPath := fmt.Sprintf("data/raft_%s.db", id)
	storage, err := NewPersistentStorage(id, dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize persistent storage: %v", err)
	}

	// Load persisted state
	currentTerm, votedFor, err := storage.LoadRaftState()
	if err != nil {
		log.Fatalf("Failed to load raft state: %v", err)
	}

	// Load snapshot metadata
	lastIncludedIndex, lastIncludedTerm, err := storage.GetSnapshotMetadata()
	if err != nil {
		log.Fatalf("Failed to load snapshot metadata: %v", err)
	}

	// Load log entries from storage (only entries after snapshot)
	logLength, err := storage.GetLogLength()
	if err != nil {
		log.Fatalf("Failed to get log length: %v", err)
	}

	var logEntries []*rpc.LogEntry
	if logLength > 0 {
		// Load entries after the snapshot
		startIndex := lastIncludedIndex + 1
		if startIndex <= logLength {
			entries, err := storage.GetLogEntries(startIndex, logLength)
			if err != nil {
				log.Fatalf("Failed to load log entries: %v", err)
			}
			logEntries = entries
		} else {
			logEntries = make([]*rpc.LogEntry, 0)
		}
	} else {
		logEntries = make([]*rpc.LogEntry, 0)
	}

	rf := &Raft{
		id:                id,
		peers:             peers,
		state:             Follower,
		votedFor:          votedFor,
		log:               logEntries,
		nextIndex:         make(map[string]int32),
		matchIndex:        make(map[string]int32),
		applyCh:           applyCh,
		currentTerm:       currentTerm,
		commitIndex:       lastIncludedIndex, // Snapshot entries are already committed
		lastApplied:       lastIncludedIndex, // Snapshot is already applied
		lastHeartbeat:     time.Now(),
		conns:             make(map[string]*grpc.ClientConn),
		storage:           storage,
		lastIncludedIndex: lastIncludedIndex,
		lastIncludedTerm:  lastIncludedTerm,
		membership:        NewMembershipConfig(peers),
	}

	// Initialize nextIndex and matchIndex for all peers
	for _, peer := range peers {
		rf.nextIndex[peer] = int32(len(rf.log)) + lastIncludedIndex + 1
		rf.matchIndex[peer] = 0
	}

	logMsg := fmt.Sprintf("Raft node initialized: term=%d, logLength=%d, votedFor=%s, snapshot.lastIncludedIndex=%d",
		currentTerm, logLength, votedFor, lastIncludedIndex)
	log.Printf("[%s] %s", id, logMsg)
	go rf.runElectionTimer()
	return rf
}

// logf prints structured logs: [node=id][term=t][state=s] msg...
func (rf *Raft) logf(format string, args ...any) {
	// Note: We might be holding the lock when calling this, or not.
	// Accessing state/term safely requires lock, but recursive locking is bad.
	// For simplicity in this project, we assume single threaded log flow or careful usage.
	// Ideally pass state/term in, or grab lock if safe.
	// Here we will just use the values we have, assuming mainly called under lock or knowing race is just logging.

	// Better: Dont call rf.GetState() if we already hold lock.
	// We'll define a version that doesn't lock if we are inside.
	// But `log` package is safe.

	// Let's just format the prefix.
	// NOTE: This access might race if not locked. Accepted for Resume Demo scope simplicity vs complexity of logger injection.
	prefix := "[" + rf.id + "]"
	log.Printf(prefix+format, args...)
}

// Helper to get structured prefix arguments safely?
// For now, simple logging consistent format:
// [node=N][term=T][state=S] Message
func (rf *Raft) info(msg string) {
	log.Printf("[node=%s][term=%d][state=%s] %s", rf.id, rf.currentTerm, rf.state, msg)
}

// GetState returns current state and term.
func (rf *Raft) GetState() (State, int32, string) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.state, rf.currentTerm, rf.leaderId
}

// Close closes the persistent storage.
func (rf *Raft) Close() error {
	if rf.storage != nil {
		return rf.storage.Close()
	}
	return nil
}

// HandleRequestVote exposes logic to Transport layer.
func (rf *Raft) HandleRequestVote(args *rpc.RequestVoteRequest) *rpc.RequestVoteResponse {
	// Forward to internal method
	reply := &rpc.RequestVoteResponse{}
	rf.RequestVote(args, reply)
	return reply
}

// HandleAppendEntries exposes logic to Transport layer.
func (rf *Raft) HandleAppendEntries(args *rpc.AppendEntriesRequest) *rpc.AppendEntriesResponse {
	reply := &rpc.AppendEntriesResponse{}
	rf.AppendEntries(args, reply)
	return reply
}

// GenerateSnapshot creates and persists a snapshot at the given index
// snapshotData should be the serialized application state (e.g., KV store)
// This should be called when the log exceeds a size threshold
func (rf *Raft) GenerateSnapshot(snapshotIndex int32, snapshotData []byte) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// Ensure snapshot index is not beyond our log
	if snapshotIndex > rf.lastIncludedIndex+int32(len(rf.log)) {
		return fmt.Errorf("snapshot index %d is beyond our log", snapshotIndex)
	}

	// Don't snapshot beyond lastApplied
	if snapshotIndex > rf.lastApplied {
		return fmt.Errorf("cannot snapshot beyond lastApplied (%d > %d)", snapshotIndex, rf.lastApplied)
	}

	// Get the term of the snapshot index
	var snapshotTerm int32
	logIndex := snapshotIndex - rf.lastIncludedIndex
	if logIndex <= 0 {
		// Entry is in previous snapshot
		snapshotTerm = rf.lastIncludedTerm
	} else if logIndex <= int32(len(rf.log)) {
		snapshotTerm = rf.log[logIndex-1].Term
	}

	// Persist snapshot
	if err := rf.storage.SaveSnapshot(snapshotIndex, snapshotTerm, snapshotData); err != nil {
		return fmt.Errorf("failed to persist snapshot: %w", err)
	}

	// Discard log entries up to and including snapshotIndex
	if logIndex > 0 {
		rf.log = rf.log[logIndex:]
	}

	// Update snapshot metadata
	rf.lastIncludedIndex = snapshotIndex
	rf.lastIncludedTerm = snapshotTerm

	rf.logf("Generated snapshot: index=%d, term=%d, dataSize=%d bytes, newLogLength=%d",
		snapshotIndex, snapshotTerm, len(snapshotData), len(rf.log))

	return nil
}

// InstallSnapshot handles InstallSnapshot RPC from leader
// Returns true if snapshot was installed, false otherwise
func (rf *Raft) InstallSnapshot(lastIncludedIndex, lastIncludedTerm int32, snapshotData []byte) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// Reject if we already have entries beyond this snapshot
	if lastIncludedIndex < rf.lastIncludedIndex {
		rf.logf("Rejected snapshot: already have index %d >= %d", rf.lastIncludedIndex, lastIncludedIndex)
		return false
	}

	// Remove log entries before lastIncludedIndex
	logStartIndex := lastIncludedIndex - rf.lastIncludedIndex
	if logStartIndex < 0 {
		logStartIndex = 0
	}

	// Keep only entries after the snapshot
	if int32(len(rf.log)) > logStartIndex {
		rf.log = rf.log[logStartIndex:]
	} else {
		rf.log = make([]*rpc.LogEntry, 0)
	}

	// Update snapshot metadata
	rf.lastIncludedIndex = lastIncludedIndex
	rf.lastIncludedTerm = lastIncludedTerm

	// Persist snapshot
	if err := rf.storage.SaveSnapshot(lastIncludedIndex, lastIncludedTerm, snapshotData); err != nil {
		rf.logf("Failed to persist snapshot: %v", err)
		return false
	}

	// Update committed/applied index to at least lastIncludedIndex
	if rf.commitIndex < lastIncludedIndex {
		rf.commitIndex = lastIncludedIndex
	}
	if rf.lastApplied < lastIncludedIndex {
		rf.lastApplied = lastIncludedIndex
	}

	rf.logf("Installed snapshot: index=%d, term=%d, dataSize=%d bytes, newLogLength=%d",
		lastIncludedIndex, lastIncludedTerm, len(snapshotData), len(rf.log))

	return true
}

// getConn returns a cached connection or dials a new one.
func (rf *Raft) getConn(peerAddr string) (*grpc.ClientConn, error) {
	rf.muConns.Lock()
	if conn, ok := rf.conns[peerAddr]; ok {
		rf.muConns.Unlock()
		return conn, nil
	}
	rf.muConns.Unlock()

	// Dial outside lock
	conn, err := grpc.Dial(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	rf.muConns.Lock()
	defer rf.muConns.Unlock()

	// Double check
	if existing, ok := rf.conns[peerAddr]; ok {
		conn.Close()
		return existing, nil
	}
	rf.conns[peerAddr] = conn
	return conn, nil
}

// Metrics support methods

// SetMetrics attaches a metrics instance to this Raft node
func (rf *Raft) SetMetrics(m interface{}) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.metrics = m
}

// GetCurrentTerm returns the current Raft term
func (rf *Raft) GetCurrentTerm() int32 {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.currentTerm
}

// GetLastApplied returns the index of the last applied entry
func (rf *Raft) GetLastApplied() int32 {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.lastApplied
}

// GetCommitIndex returns the current commit index
func (rf *Raft) GetCommitIndex() int32 {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.commitIndex
}

// GetLastIncludedIndex returns the last index included in snapshot
func (rf *Raft) GetLastIncludedIndex() int32 {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.lastIncludedIndex
}

// IsLeader returns true if this node is the current leader
func (rf *Raft) IsLeader() bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.state == Leader
}
