package raft

import (
	"fmt"
	"log"
	"sync"
	"time"

	"distributed-kv-raft/rpc"
)

// PeerState represents the replication state of a peer
type PeerState int

const (
	PeerStateUnknown PeerState = iota
	PeerStateReplicating
	PeerStateConfirmed
)

func (ps PeerState) String() string {
	switch ps {
	case PeerStateReplicating:
		return "Replicating"
	case PeerStateConfirmed:
		return "Confirmed"
	default:
		return "Unknown"
	}
}

// PeerInfo stores information about a peer node
type PeerInfo struct {
	ID        string
	Address   string
	State     PeerState
	LastSeen  time.Time
	Added     time.Time
}

// MembershipConfig manages cluster membership
type MembershipConfig struct {
	mu    sync.RWMutex
	peers map[string]*PeerInfo
}

// NewMembershipConfig creates a new membership configuration
func NewMembershipConfig(initialPeers []string) *MembershipConfig {
	mc := &MembershipConfig{
		peers: make(map[string]*PeerInfo),
	}
	
	// Convert initial peer addresses to PeerInfo
	for _, peerAddr := range initialPeers {
		info := &PeerInfo{
			ID:       fmt.Sprintf("peer-%s", peerAddr),
			Address:  peerAddr,
			State:    PeerStateConfirmed,
			LastSeen: time.Now(),
			Added:    time.Now(),
		}
		mc.peers[info.ID] = info
	}
	
	return mc
}

// GetPeers returns a copy of all peers
func (mc *MembershipConfig) GetPeers() map[string]*PeerInfo {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	result := make(map[string]*PeerInfo)
	for id, info := range mc.peers {
		copied := *info
		result[id] = &copied
	}
	return result
}

// GetPeerAddresses returns just the addresses
func (mc *MembershipConfig) GetPeerAddresses() []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	var addrs []string
	for _, info := range mc.peers {
		addrs = append(addrs, info.Address)
	}
	return addrs
}

// AddPeer adds a new peer to the cluster
// Returns error if peer already exists or if not leader
func (mc *MembershipConfig) AddPeer(id, address string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// Check if peer already exists
	for _, info := range mc.peers {
		if info.ID == id || info.Address == address {
			return fmt.Errorf("peer already exists: %s (%s)", id, address)
		}
	}
	
	// Add new peer in unconfirmed state
	info := &PeerInfo{
		ID:       id,
		Address:  address,
		State:    PeerStateReplicating,
		LastSeen: time.Now(),
		Added:    time.Now(),
	}
	mc.peers[id] = info
	log.Printf("[Membership] Added peer: %s (%s) in %s state", id, address, info.State)
	
	return nil
}

// ConfirmPeer marks a peer as confirmed (fully replicated)
func (mc *MembershipConfig) ConfirmPeer(id string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	info, exists := mc.peers[id]
	if !exists {
		return fmt.Errorf("peer not found: %s", id)
	}
	
	if info.State != PeerStateReplicating {
		return fmt.Errorf("peer %s not in replicating state", id)
	}
	
	info.State = PeerStateConfirmed
	log.Printf("[Membership] Confirmed peer: %s", id)
	
	return nil
}

// RemovePeer removes a peer from the cluster
func (mc *MembershipConfig) RemovePeer(id string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if _, exists := mc.peers[id]; !exists {
		return fmt.Errorf("peer not found: %s", id)
	}
	
	delete(mc.peers, id)
	log.Printf("[Membership] Removed peer: %s", id)
	
	return nil
}

// GetConfirmedPeerCount returns number of confirmed peers
func (mc *MembershipConfig) GetConfirmedPeerCount() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	count := 0
	for _, info := range mc.peers {
		if info.State == PeerStateConfirmed {
			count++
		}
	}
	return count
}

// GetMajority returns the number needed for quorum (confirmed peers)
func (mc *MembershipConfig) GetMajority() int {
	confirmedCount := mc.GetConfirmedPeerCount()
	if confirmedCount == 0 {
		return 1
	}
	return confirmedCount/2 + 1
}

// UpdateLastSeen updates the last seen time for a peer
func (mc *MembershipConfig) UpdateLastSeen(id string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if info, exists := mc.peers[id]; exists {
		info.LastSeen = time.Now()
	}
}

// IsHealthy checks if a peer is responding
func (mc *MembershipConfig) IsHealthy(id string, timeout time.Duration) bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	info, exists := mc.peers[id]
	if !exists {
		return false
	}
	
	return time.Since(info.LastSeen) < timeout
}

// HasQuorum checks if we have a healthy quorum
func (mc *MembershipConfig) HasQuorum(timeout time.Duration) bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	confirmedCount := 0
	healthyCount := 0
	
	for _, info := range mc.peers {
		if info.State == PeerStateConfirmed {
			confirmedCount++
			if time.Since(info.LastSeen) < timeout {
				healthyCount++
			}
		}
	}
	
	if confirmedCount == 0 {
		return true // Single node cluster
	}
	
	return healthyCount >= (confirmedCount/2 + 1)
}

// MembershipChangeEntry represents a committed membership change in the log
type MembershipChangeEntry struct {
	NodeID  string    // Node identifier
	Address string    // Node address (e.g., "10.0.1.10:9000")
	Action  string    // "ADD" or "REMOVE"
	Term    int32     // Term when committed
	Index   int32     // Log index
	Applied time.Time // When applied to state machine
}

// ApplyMembershipChange applies a membership change entry to the cluster
func (rf *Raft) ApplyMembershipChange(nodeID, address, action string) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	
	// Only leader can apply membership changes
	if rf.state != Leader {
		return fmt.Errorf("only leader can apply membership changes")
	}
	
	switch action {
	case "ADD":
		return rf.membership.AddPeer(nodeID, address)
	case "REMOVE":
		return rf.membership.RemovePeer(nodeID)
	default:
		return fmt.Errorf("unknown membership action: %s", action)
	}
}

// AppendMembershipChange appends a membership change to the log
// Returns the index and term of the new entry
func (rf *Raft) AppendMembershipChange(nodeID, address, action string) (int32, int32, error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	
	// Only leader can append
	if rf.state != Leader {
		return 0, 0, fmt.Errorf("not leader")
	}
	
	// Create membership change command
	command := fmt.Sprintf("MEMBERSHIP %s %s %s", nodeID, address, action)
	
	// Append to log
	entry := &rpc.LogEntry{
		Term:    rf.currentTerm,
		Index:   int32(len(rf.log)) + rf.lastIncludedIndex + 1,
		Command: command,
	}
	
	rf.log = append(rf.log, entry)
	
	// Persist to storage
	if rf.storage != nil {
		rf.storage.AppendLogEntry(entry)
	}
	
	log.Printf("[%s][term=%d] Appended membership change: %s", rf.id, rf.currentTerm, command)
	
	return entry.Index, entry.Term, nil
}

// GetMembershipInfo returns current membership information
func (rf *Raft) GetMembershipInfo() map[string]interface{} {
	rf.mu.Lock()
	peers := rf.membership.GetPeers()
	confirmedCount := rf.membership.GetConfirmedPeerCount()
	rf.mu.Unlock()
	
	peerList := make([]map[string]interface{}, 0)
	for id, info := range peers {
		peerList = append(peerList, map[string]interface{}{
			"id":         id,
			"address":    info.Address,
			"state":      info.State.String(),
			"last_seen":  info.LastSeen,
			"added":      info.Added,
		})
	}
	
	return map[string]interface{}{
		"node_id":          rf.id,
		"peers":            peerList,
		"confirmed_count":  confirmedCount,
		"majority":         rf.membership.GetMajority(),
		"has_quorum":       rf.membership.HasQuorum(10 * time.Second),
	}
}
