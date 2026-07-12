package raft

import (
	"context"
	"time"

	"distributed-kv-raft/rpc"
)

// runHeartbeatLoop sends AppendEntries to all peers periodically.
func (rf *Raft) runHeartbeatLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		rf.mu.Lock()
		if rf.state != Leader {
			rf.mu.Unlock()
			return
		}
		rf.mu.Unlock()

		// Send to all peers
		for _, peer := range rf.peers {
			go func(peerAddr string) {
				rf.mu.Lock()
				if rf.state != Leader {
					rf.mu.Unlock()
					return
				}
				nextIdx := rf.nextIndex[peerAddr]
				lastIncludedIdx := rf.lastIncludedIndex

				// Check if follower needs a snapshot
				if nextIdx <= lastIncludedIdx {
					// TODO: Send snapshot via InstallSnapshot RPC
					// For now, just backoff
					if rf.nextIndex[peerAddr] > 1 {
						rf.nextIndex[peerAddr]--
					}
					rf.mu.Unlock()
					return
				}

				// Convert absolute log index to relative index in rf.log
				prevLogIndex := nextIdx - 1
				prevLogTerm := int32(0)

				// Get prevLogTerm
				if prevLogIndex == lastIncludedIdx {
					prevLogTerm = rf.lastIncludedTerm
				} else if prevLogIndex > lastIncludedIdx {
					relIdx := prevLogIndex - lastIncludedIdx - 1
					if relIdx >= 0 && relIdx < int32(len(rf.log)) {
						prevLogTerm = rf.log[relIdx].Term
					}
				}

				// Get entries to send (after prevLogIndex)
				entries := make([]*rpc.LogEntry, 0)
				if nextIdx <= lastIncludedIdx+int32(len(rf.log)) {
					relStartIdx := nextIdx - lastIncludedIdx - 1
					if relStartIdx >= 0 && relStartIdx < int32(len(rf.log)) {
						entries = rf.log[relStartIdx:]
					}
				}

				term := rf.currentTerm
				leaderId := rf.leaderId
				commitIndex := rf.commitIndex

				args := &rpc.AppendEntriesRequest{
					Term:         term,
					LeaderId:     leaderId,
					PrevLogIndex: prevLogIndex,
					PrevLogTerm:  prevLogTerm,
					Entries:      entries,
					LeaderCommit: commitIndex,
				}
				rf.mu.Unlock()

				// We need a reply struct
				// Note: rf.sendAppendEntries signature change needed to return (reply, error) or handle internally?
				// Existing signature: func (rf *Raft) sendAppendEntries(peerId string, args *rpc.AppendEntriesRequest, reply *rpc.AppendEntriesResponse) bool
				// We will stick to the existing signature style but implement logic inside using gRPC.

				reply := &rpc.AppendEntriesResponse{}
				if rf.sendAppendEntries(peerAddr, args, reply) {
					rf.mu.Lock()
					defer rf.mu.Unlock()
					if rf.state != Leader {
						return
					}
					if reply.Term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.state = Follower
						rf.votedFor = ""
						// Persist state change
						if err := rf.storage.SaveRaftState(rf.currentTerm, rf.votedFor); err != nil {
							rf.logf("Failed to persist raft state: %v", err)
						}
						return
					}
					if reply.Success {
						match := args.PrevLogIndex + int32(len(args.Entries))
						rf.logf("Replication success to %s: matchIndex=%d", peerAddr, match)
						rf.matchIndex[peerAddr] = match
						rf.nextIndex[peerAddr] = match + 1
						rf.updateCommitIndex()
					} else {
						// Back off nextIndex
						if rf.nextIndex[peerAddr] > 1 {
							rf.nextIndex[peerAddr]--
						}
					}
				}
			}(peer)
		}

		<-ticker.C
	}
}

// sendAppendEntries implements RPC client call.
func (rf *Raft) sendAppendEntries(peerAddr string, args *rpc.AppendEntriesRequest, reply *rpc.AppendEntriesResponse) bool {
	// rf.logf("Sending AppendEntries to %s", peerAddr) // Check spam

	conn, err := rf.getConn(peerAddr)
	if err != nil {
		// rf.logf("AppendEntries dial failed: %v", err)
		return false
	}
	// defer conn.Close()

	client := rpc.NewRaftServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10s timeout
	defer cancel()

	resp, err := client.AppendEntries(ctx, args)
	if err != nil {
		rf.logf("AppendEntries RPC failed to %s: %v", peerAddr, err)
		return false
	}

	reply.Term = resp.Term
	reply.Success = resp.Success
	return true
}

// AppendEntries handler.
func (rf *Raft) AppendEntries(args *rpc.AppendEntriesRequest, reply *rpc.AppendEntriesResponse) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// rf.logf("AppendEntries RECEIVED from=%s term=%d", args.LeaderId, args.Term)

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.state = Follower
		rf.votedFor = ""
		// Persist state change
		if err := rf.storage.SaveRaftState(rf.currentTerm, rf.votedFor); err != nil {
			rf.logf("Failed to persist raft state: %v", err)
		}
	}

	reply.Term = rf.currentTerm
	reply.Success = false

	if args.Term < rf.currentTerm {
		return
	}

	// Reset election timer since we heard from valid leader
	rf.lastHeartbeat = time.Now()
	rf.leaderId = args.LeaderId

	// Log consistency check
	// PrevLogIndex must match our state
	if args.PrevLogIndex > 0 {
		// Check if prevLogIndex is in the snapshot
		if args.PrevLogIndex == rf.lastIncludedIndex {
			// Verify prevLogTerm matches snapshot
			if args.PrevLogTerm != rf.lastIncludedTerm {
				return
			}
		} else if args.PrevLogIndex > rf.lastIncludedIndex {
			// prevLogIndex is in our log
			relIdx := args.PrevLogIndex - rf.lastIncludedIndex - 1
			if relIdx < 0 || relIdx >= int32(len(rf.log)) {
				return
			}
			if rf.log[relIdx].Term != args.PrevLogTerm {
				// Conflict: delete everything from here
				rf.log = rf.log[:relIdx]
				// Persist log truncation
				for i := len(rf.log); i >= 1; i-- {
					if err := rf.storage.TruncateLog(int32(i) + rf.lastIncludedIndex); err != nil {
						rf.logf("Failed to truncate log at %d: %v", i, err)
					}
				}
				return
			}
		} else {
			// prevLogIndex is before our snapshot - reject
			return
		}
	}

	// Append new entries
	// Note: Be careful not to truncate correctly matching entries if we receive a stale packet?
	// Raft paper says: "If an existing entry conflicts with a new one... delete the existing entry and all that follow it."
	for i, entry := range args.Entries {
		absIdx := args.PrevLogIndex + 1 + int32(i)
		relIdx := absIdx - rf.lastIncludedIndex - 1

		if relIdx < 0 {
			// Entry is in snapshot, skip
			continue
		}

		if relIdx < int32(len(rf.log)) {
			// Entry exists in our log
			if rf.log[relIdx].Term != entry.Term {
				// Conflict: delete from here onward
				rf.log = rf.log[:relIdx]
				rf.log = append(rf.log, entry)
			}
		} else {
			// Append new entry
			rf.log = append(rf.log, entry)
		}
		// Persist new entries
		if err := rf.storage.AppendLogEntry(entry); err != nil {
			rf.logf("Failed to persist log entry: %v", err)
		}
	}

	// Update commit index
	if args.LeaderCommit > rf.commitIndex {
		lastNewIndex := args.PrevLogIndex + int32(len(args.Entries))
		if args.LeaderCommit < lastNewIndex {
			rf.commitIndex = args.LeaderCommit
		} else {
			rf.commitIndex = lastNewIndex
		}
		rf.applyLogs()
	}

	reply.Success = true
}

func (rf *Raft) updateCommitIndex() {
	// Calculate N such that majority of matchIndex[i] >= N
	// All indices are absolute (including snapshot)
	matchIndexes := make([]int32, 0, len(rf.peers)+1)
	// Leader's matchIndex is at the end of log (absolute index)
	leaderMatchIndex := rf.lastIncludedIndex + int32(len(rf.log))
	matchIndexes = append(matchIndexes, leaderMatchIndex)
	for _, peer := range rf.peers {
		matchIndexes = append(matchIndexes, rf.matchIndex[peer])
	}
	rf.logf("updateCommitIndex: matchIndexes=%v commitIndex=%d", matchIndexes, rf.commitIndex)

	// Simple O(N^2) sort for small N (3-5 nodes)
	for i := 0; i < len(matchIndexes); i++ {
		for j := i + 1; j < len(matchIndexes); j++ {
			if matchIndexes[i] > matchIndexes[j] {
				matchIndexes[i], matchIndexes[j] = matchIndexes[j], matchIndexes[i]
			}
		}
	}

	// Majority index is at position (N-1)/2 ?? No.
	// 3 nodes: [m0, m1, m2] sorted. Majority needs 2. Index 1?
	// If [1, 5, 5], majority >= 5. Median.
	// len=3, mid=1. 3/2 = 1.
	// 5 nodes: [1,1,3,5,5], majority >= 3. len=5, mid=2. 5/2 = 2.

	n := len(matchIndexes)
	majorityIndex := matchIndexes[n/2] // This is effectively the median if indices are sorted

	// Check term condition: log[N].term == currentTerm
	if majorityIndex > rf.commitIndex {
		// Get the term of majorityIndex entry
		var entryTerm int32
		if majorityIndex == rf.lastIncludedIndex {
			// Entry is the snapshot
			entryTerm = rf.lastIncludedTerm
		} else if majorityIndex > rf.lastIncludedIndex {
			// Entry is in log
			relIdx := majorityIndex - rf.lastIncludedIndex - 1
			if relIdx >= 0 && relIdx < int32(len(rf.log)) {
				entryTerm = rf.log[relIdx].Term
			}
		}

		// Only advance commitIndex if entry is from current term
		if entryTerm == rf.currentTerm {
			rf.commitIndex = majorityIndex
			rf.logf("CommitIndex updated to %d", rf.commitIndex)
			rf.applyLogs()
		}
	}
}

func (rf *Raft) applyLogs() {
	for rf.lastApplied < rf.commitIndex {
		rf.lastApplied++
		
		// Convert absolute index to relative index in log
		relIdx := rf.lastApplied - rf.lastIncludedIndex - 1
		if relIdx < 0 || relIdx >= int32(len(rf.log)) {
			// Entry is in snapshot, already applied
			continue
		}

		entry := rf.log[relIdx]
		rf.logf("Applying entry: index=%d term=%d cmd=%s", entry.Index, entry.Term, entry.Command)
		rf.applyCh <- entry

		// Check if we should generate a snapshot (every 10 entries)
		if rf.lastApplied%10 == 0 && rf.state == Leader {
			// Leader can trigger snapshot generation
			// TODO: Call snapshot generation from state machine
		}
	}
}

// ShouldGenerateSnapshot checks if we should generate a snapshot
// Returns true if log size exceeds threshold (10 entries)
func (rf *Raft) ShouldGenerateSnapshot() bool {
	return int32(len(rf.log)) > 10
}

// GetLogLength returns the total length of the log (including snapshot)
func (rf *Raft) GetLogLength() int32 {
	return rf.lastIncludedIndex + int32(len(rf.log))
}

// StartCommand is called by client to propose a new command (Log Replication start).
func (rf *Raft) StartCommand(command string) (int32, int32, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.state != Leader {
		return -1, -1, false
	}
	// Use absolute index (account for snapshot)
	absIndex := rf.lastIncludedIndex + int32(len(rf.log)) + 1
	term := rf.currentTerm
	entry := &rpc.LogEntry{
		Index:   absIndex,
		Term:    term,
		Command: command,
	}
	rf.log = append(rf.log, entry)
	rf.logf("StartCommand: index=%d term=%d cmd=%s", absIndex, term, command)

	// Persist new log entry
	if err := rf.storage.AppendLogEntry(entry); err != nil {
		rf.logf("Failed to persist log entry: %v", err)
		// Still return success since it's in memory; persistence is a safety feature
	}

	// Trigger heartbeat immediately to speed up replication?
	// Optional but good for responsiveness.
	// For now rely on ticker.

	return absIndex, term, true
}
