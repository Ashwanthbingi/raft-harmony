package raft

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"distributed-kv-raft/rpc"
	"go.etcd.io/bbolt"
)

// PersistentStorage provides durable storage for Raft state using BoltDB
type PersistentStorage struct {
	mu       sync.RWMutex
	db       *bbolt.DB
	dbPath   string
	nodeID   string
	buckets  persistenceBuckets
}

// persistenceBuckets defines the bucket names in BoltDB
type persistenceBuckets struct {
	raftState string // Stores current term and votedFor
	logData   string // Stores log entries
	metadata  string // Stores metadata like lastIndex
}

// NewPersistentStorage creates a new persistent storage instance
func NewPersistentStorage(nodeID string, dbPath string) (*PersistentStorage, error) {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ps := &PersistentStorage{
		db:     db,
		dbPath: dbPath,
		nodeID: nodeID,
		buckets: persistenceBuckets{
			raftState: "raft_state",
			logData:   "log_entries",
			metadata:  "metadata",
		},
	}

	// Initialize buckets
	if err := ps.initializeBuckets(); err != nil {
		db.Close()
		return nil, err
	}

	return ps, nil
}

// initializeBuckets creates necessary buckets if they don't exist
func (ps *PersistentStorage) initializeBuckets() error {
	return ps.db.Update(func(tx *bbolt.Tx) error {
		for _, bucketName := range []string{ps.buckets.raftState, ps.buckets.logData, ps.buckets.metadata} {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
				return err
			}
		}
		return nil
	})
}

// RaftState represents the persistent Raft state
type RaftState struct {
	CurrentTerm int32  `json:"current_term"`
	VotedFor    string `json:"voted_for"`
}

// SaveRaftState persists the current term and voted for information
func (ps *PersistentStorage) SaveRaftState(currentTerm int32, votedFor string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	state := RaftState{
		CurrentTerm: currentTerm,
		VotedFor:    votedFor,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal raft state: %w", err)
	}

	return ps.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ps.buckets.raftState))
		if err := b.Put([]byte("state"), data); err != nil {
			return err
		}
		log.Printf("[%s] Saved raft state: term=%d, votedFor=%s", ps.nodeID, currentTerm, votedFor)
		return nil
	})
}

// LoadRaftState loads the persisted Raft state
func (ps *PersistentStorage) LoadRaftState() (currentTerm int32, votedFor string, err error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var state RaftState

	err = ps.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ps.buckets.raftState))
		data := b.Get([]byte("state"))

		if data == nil {
			// First start - no state saved
			log.Printf("[%s] No raft state found - first start", ps.nodeID)
			return nil
		}

		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("failed to unmarshal raft state: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, "", err
	}

	return state.CurrentTerm, state.VotedFor, nil
}

// AppendLogEntry appends a new log entry and persists it
func (ps *PersistentStorage) AppendLogEntry(entry *rpc.LogEntry) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return ps.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ps.buckets.logData))
		meta := tx.Bucket([]byte(ps.buckets.metadata))

		// Get the next index
		nextIndexBytes := meta.Get([]byte("next_index"))
		nextIndex := int32(1)
		if nextIndexBytes != nil {
			fmt.Sscanf(string(nextIndexBytes), "%d", &nextIndex)
		}

		// Serialize the entry
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		// Store with key: entry_<index>
		key := fmt.Sprintf("entry_%d", nextIndex)
		if err := b.Put([]byte(key), data); err != nil {
			return err
		}

		// Update next_index
		if err := meta.Put([]byte("next_index"), []byte(fmt.Sprintf("%d", nextIndex+1))); err != nil {
			return err
		}

		log.Printf("[%s] Appended log entry: index=%d, term=%d, command=%s", ps.nodeID, nextIndex, entry.Term, entry.Command)
		return nil
	})
}

// GetLogEntry retrieves a specific log entry by index (1-indexed)
func (ps *PersistentStorage) GetLogEntry(index int32) (*rpc.LogEntry, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var entry *rpc.LogEntry

	err := ps.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ps.buckets.logData))
		key := fmt.Sprintf("entry_%d", index)
		data := b.Get([]byte(key))

		if data == nil {
			return nil // Entry doesn't exist
		}

		entry = &rpc.LogEntry{}
		if err := json.Unmarshal(data, entry); err != nil {
			return fmt.Errorf("failed to unmarshal log entry: %w", err)
		}

		return nil
	})

	return entry, err
}

// GetLogEntries retrieves all log entries from startIndex to endIndex (1-indexed, inclusive)
func (ps *PersistentStorage) GetLogEntries(startIndex, endIndex int32) ([]*rpc.LogEntry, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var entries []*rpc.LogEntry

	err := ps.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ps.buckets.logData))
		meta := tx.Bucket([]byte(ps.buckets.metadata))

		// Get total number of entries
		nextIndexBytes := meta.Get([]byte("next_index"))
		nextIndex := int32(1)
		if nextIndexBytes != nil {
			fmt.Sscanf(string(nextIndexBytes), "%d", &nextIndex)
		}

		// Ensure valid range
		if startIndex < 1 {
			startIndex = 1
		}
		if endIndex >= nextIndex {
			endIndex = nextIndex - 1
		}
		if startIndex > endIndex {
			return nil
		}

		// Retrieve entries in range
		for i := startIndex; i <= endIndex; i++ {
			key := fmt.Sprintf("entry_%d", i)
			data := b.Get([]byte(key))
			if data != nil {
				entry := &rpc.LogEntry{}
				if err := json.Unmarshal(data, entry); err != nil {
					return err
				}
				entries = append(entries, entry)
			}
		}

		return nil
	})

	return entries, err
}

// GetLastLogIndex returns the index of the last log entry
func (ps *PersistentStorage) GetLastLogIndex() (int32, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var lastIndex int32

	err := ps.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte(ps.buckets.metadata))
		nextIndexBytes := meta.Get([]byte("next_index"))

		if nextIndexBytes != nil {
			fmt.Sscanf(string(nextIndexBytes), "%d", &lastIndex)
			lastIndex-- // Convert from next_index to last_index
		}

		return nil
	})

	return lastIndex, err
}

// TruncateLog removes all log entries from startIndex onwards
func (ps *PersistentStorage) TruncateLog(startIndex int32) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return ps.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(ps.buckets.logData))
		meta := tx.Bucket([]byte(ps.buckets.metadata))

		// Get total number of entries
		nextIndexBytes := meta.Get([]byte("next_index"))
		nextIndex := int32(1)
		if nextIndexBytes != nil {
			fmt.Sscanf(string(nextIndexBytes), "%d", &nextIndex)
		}

		// Delete all entries from startIndex onwards
		for i := startIndex; i < nextIndex; i++ {
			key := fmt.Sprintf("entry_%d", i)
			if err := b.Delete([]byte(key)); err != nil {
				return err
			}
		}

		// Update next_index
		if err := meta.Put([]byte("next_index"), []byte(fmt.Sprintf("%d", startIndex))); err != nil {
			return err
		}

		log.Printf("[%s] Truncated log from index %d", ps.nodeID, startIndex)
		return nil
	})
}

// GetLogLength returns the total number of log entries
func (ps *PersistentStorage) GetLogLength() (int32, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var length int32

	err := ps.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte(ps.buckets.metadata))
		nextIndexBytes := meta.Get([]byte("next_index"))

		if nextIndexBytes != nil {
			fmt.Sscanf(string(nextIndexBytes), "%d", &length)
			length-- // Convert from next_index to length
		}

		return nil
	})

	return length, err
}

// Close closes the database connection
func (ps *PersistentStorage) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}

// Stats returns storage statistics
func (ps *PersistentStorage) Stats() map[string]interface{} {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	stats := make(map[string]interface{})

	ps.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte(ps.buckets.metadata))
		raftState := tx.Bucket([]byte(ps.buckets.raftState))

		nextIndexBytes := meta.Get([]byte("next_index"))
		logLength := int32(0)
		if nextIndexBytes != nil {
			fmt.Sscanf(string(nextIndexBytes), "%d", &logLength)
			logLength--
		}

		stateData := raftState.Get([]byte("state"))
		var state RaftState
		if stateData != nil {
			json.Unmarshal(stateData, &state)
		}

		stats["log_entries"] = logLength
		stats["current_term"] = state.CurrentTerm
		stats["voted_for"] = state.VotedFor
		stats["db_path"] = ps.dbPath

		return nil
	})

	return stats
}

// Snapshot represents a Raft snapshot
type Snapshot struct {
	LastIncludedIndex int32  `json:"last_included_index"` // last log index included in snapshot
	LastIncludedTerm  int32  `json:"last_included_term"`  // term of last_included_index
	Data              []byte `json:"data"`                // serialized state machine data
}

// SaveSnapshot persists a snapshot to disk
func (ps *PersistentStorage) SaveSnapshot(lastIncludedIndex, lastIncludedTerm int32, data []byte) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	snapshot := Snapshot{
		LastIncludedIndex: lastIncludedIndex,
		LastIncludedTerm:  lastIncludedTerm,
		Data:              data,
	}

	return ps.db.Update(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte(ps.buckets.metadata))
		
		snapshotData, err := json.Marshal(snapshot)
		if err != nil {
			return fmt.Errorf("failed to marshal snapshot: %w", err)
		}

		if err := meta.Put([]byte("snapshot"), snapshotData); err != nil {
			return err
		}

		log.Printf("[%s] Saved snapshot: lastIncludedIndex=%d, dataSize=%d bytes", 
			ps.nodeID, lastIncludedIndex, len(data))
		return nil
	})
}

// LoadSnapshot loads the latest snapshot from disk
func (ps *PersistentStorage) LoadSnapshot() (*Snapshot, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var snapshot *Snapshot

	err := ps.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte(ps.buckets.metadata))
		snapshotData := meta.Get([]byte("snapshot"))

		if snapshotData == nil {
			// No snapshot yet
			log.Printf("[%s] No snapshot found - first start", ps.nodeID)
			return nil
		}

		snapshot = &Snapshot{}
		if err := json.Unmarshal(snapshotData, snapshot); err != nil {
			return fmt.Errorf("failed to unmarshal snapshot: %w", err)
		}

		return nil
	})

	return snapshot, err
}

// GetSnapshotMetadata returns snapshot metadata without loading full data
func (ps *PersistentStorage) GetSnapshotMetadata() (lastIncludedIndex, lastIncludedTerm int32, err error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	err = ps.db.View(func(tx *bbolt.Tx) error {
		meta := tx.Bucket([]byte(ps.buckets.metadata))
		snapshotData := meta.Get([]byte("snapshot"))

		if snapshotData == nil {
			return nil
		}

		var snapshot Snapshot
		if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
			return err
		}

		lastIncludedIndex = snapshot.LastIncludedIndex
		lastIncludedTerm = snapshot.LastIncludedTerm
		return nil
	})

	return
}

