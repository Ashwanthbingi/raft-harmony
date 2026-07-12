package raft

import (
	"context"
	"math/rand"
	"time"

	"distributed-kv-raft/rpc"
)

const (
	electionTimeoutMin = 3000 * time.Millisecond
	electionTimeoutMax = 6000 * time.Millisecond
	heartbeatInterval  = 1000 * time.Millisecond
)

// StartElection begins the election process.
func (rf *Raft) StartElection() {
	rf.mu.Lock()
	rf.state = Candidate
	rf.currentTerm++
	rf.votedFor = rf.id
	term := rf.currentTerm
	
	// Persist state
	if err := rf.storage.SaveRaftState(rf.currentTerm, rf.votedFor); err != nil {
		rf.logf("Failed to persist raft state during election: %v", err)
	}
	
	// Get last log info
	var lastLogIndex, lastLogTerm int32
	if len(rf.log) > 0 {
		lastEntry := rf.log[len(rf.log)-1]
		lastLogIndex = lastEntry.Index
		lastLogTerm = lastEntry.Term
	}
	rf.mu.Unlock()

	// Reset timer
	rf.resetElectionTimer()

	votes := make(map[string]bool)
	votes[rf.id] = true
	votesNeeded := (len(rf.peers)+1)/2 + 1
	rf.logf("Starting election for term %d. Need %d votes.", term, votesNeeded)

	for _, peer := range rf.peers {
		go func(peerAddr string) {
			args := &rpc.RequestVoteRequest{
				Term:         term,
				CandidateId:  rf.id,
				LastLogIndex: lastLogIndex,
				LastLogTerm:  lastLogTerm,
			}

			// Call new signature
			reply, err := rf.sendRequestVote(peerAddr, args)
			if err != nil {
				rf.logf("RequestVote to %s failed: %v", peerAddr, err)
				return
			}

			rf.mu.Lock()
			defer rf.mu.Unlock()

			if rf.state != Candidate {
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
			if reply.VoteGranted {
				votes[peerAddr] = true
				rf.logf("Vote received from %s. Total: %d/%d", peerAddr, len(votes), len(rf.peers)+1)
				if len(votes) >= votesNeeded {
					rf.becomeLeader()
				}
			}
		}(peer)
	}
}

// becomeLeader transitions to Leader state.
func (rf *Raft) becomeLeader() {
	if rf.state != Candidate {
		return
	}
	rf.state = Leader
	rf.leaderId = rf.id
	rf.logf("Became Leader for term %d", rf.currentTerm)

	// Reinitialize volatile state
	// nextIndex should be absolute index (account for snapshot)
	for _, peer := range rf.peers {
		rf.nextIndex[peer] = rf.lastIncludedIndex + int32(len(rf.log)) + 1 // optimistic
		rf.matchIndex[peer] = 0
	}

	// Start heartbeat immediately
	go rf.runHeartbeatLoop()
}

// resetElectionTimer logic moved to end of file

// sendRequestVote implements RPC client call.
func (rf *Raft) sendRequestVote(peerAddr string, args *rpc.RequestVoteRequest) (*rpc.RequestVoteResponse, error) {
	rf.logf("Sending RequestVote to %s for term %d", peerAddr, args.Term)

	conn, err := rf.getConn(peerAddr)
	if err != nil {
		return nil, err
	}
	// Connection is pooled, do not close.
	// defer conn.Close()

	client := rpc.NewRaftServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return client.RequestVote(ctx, args)
}

// RequestVote handler
func (rf *Raft) RequestVote(args *rpc.RequestVoteRequest, reply *rpc.RequestVoteResponse) {
	rf.logf("RequestVote RECEIVED from=%s term=%d", args.CandidateId, args.Term)
	rf.mu.Lock()
	defer rf.mu.Unlock()

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
	reply.VoteGranted = false

	if args.Term < rf.currentTerm {
		return
	}

	// Check if already voted
	if (rf.votedFor == "" || rf.votedFor == args.CandidateId) && rf.isLogUpToDate(args.LastLogIndex, args.LastLogTerm) {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		rf.state = Follower // reset to follower if we grant vote
		rf.lastHeartbeat = time.Now()
		// Persist vote
		if err := rf.storage.SaveRaftState(rf.currentTerm, rf.votedFor); err != nil {
			rf.logf("Failed to persist raft state: %v", err)
		}
	}
}

func (rf *Raft) isLogUpToDate(cIndex, cTerm int32) bool {
	var lastIndex, lastTerm int32
	if len(rf.log) > 0 {
		last := rf.log[len(rf.log)-1]
		lastIndex = last.Index
		lastTerm = last.Term
	}
	if cTerm != lastTerm {
		return cTerm > lastTerm
	}
	return cIndex >= lastIndex
}

// runElectionTimer is the main loop that checks if we should start an election.
func (rf *Raft) runElectionTimer() {
	rf.logf("Election timer started")
	for {
		timeout := electionTimeoutMin + time.Duration(rand.Int63n(int64(electionTimeoutMax-electionTimeoutMin)))
		time.Sleep(timeout)

		rf.mu.Lock()
		if rf.state == Leader {
			rf.mu.Unlock()
			continue
		}

		// If we haven't heard from leader recently (implied by sleep finishing without being reset? No, sleep triggers anyway)
		// Wait, simple sleep loop doesn't work well if we want to "reset" it on heartbeat.
		// Better pattern: use a timestamp of last heartbeat.

		if time.Since(rf.lastHeartbeat) > timeout {
			rf.logf("Election timer expired. Starting election.") // Optional log
			// StartElection in current code grabs lock. So we unlock before calling it.
			rf.mu.Unlock()
			rf.StartElection()
			// Fix #1: Reset timer immediately to prevent tight loop
			rf.resetElectionTimer()
		} else {
			rf.mu.Unlock()
		}
	}
}

// resetElectionTimer updates the last heartbeat time.
func (rf *Raft) resetElectionTimer() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.lastHeartbeat = time.Now()
}
