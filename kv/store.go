package kv

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu   sync.RWMutex
	data map[string]string

	// Metrics for observability
	metrics interface{} // *metrics.RaftMetrics (avoid circular import)
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

// SetMetrics attaches a metrics instance to this store
func (s *Store) SetMetrics(m interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = m
}

func (s *Store) Get(key string) (string, bool) {
	start := time.Now()

	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]

	// Record metrics if available
	latency := time.Since(start).Seconds()
	if m, ok := s.metrics.(interface {
		RecordGetMetrics(float64)
	}); ok {
		m.RecordGetMetrics(latency)
	}

	return val, ok
}

func (s *Store) Set(key, value string) {
	start := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value

	// Record metrics if available
	latency := time.Since(start).Seconds()
	if m, ok := s.metrics.(interface {
		RecordSetMetrics(float64)
	}); ok {
		m.RecordSetMetrics(latency)
	}
}

// Delete removes a key from the store
func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[key]; ok {
		delete(s.data, key)
		return true
	}
	return false
}

// Serialize converts store state to JSON for snapshots
func (s *Store) Serialize() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, _ := json.Marshal(s.data)
	return data
}

// Restore loads state from snapshot JSON
func (s *Store) Restore(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.data)
}

// GetPrefix returns all keys with given prefix
func (s *Store) GetPrefix(prefix string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range s.data {
		if strings.HasPrefix(k, prefix) {
			result[k] = v
		}
	}
	return result
}

// GetRange returns all keys in the range [start, end)
func (s *Store) GetRange(start, end string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range s.data {
		if k >= start && (end == "" || k < end) {
			result[k] = v
		}
	}
	return result
}

// ListKeys returns all keys in sorted order
func (s *Store) ListKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ListKeysWithLimit returns keys in sorted order with limit and offset
func (s *Store) ListKeysWithLimit(limit, offset int) ([]string, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	total := len(keys)
	if offset >= total {
		return []string{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return keys[offset:end], total
}

// GetMultiple returns values for multiple keys
func (s *Store) GetMultiple(keys []string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)
	for _, k := range keys {
		if v, ok := s.data[k]; ok {
			result[k] = v
		}
	}
	return result
}

// SetMultiple sets multiple key-value pairs
// Returns map of results: key -> (success bool, error string)
func (s *Store) SetMultiple(kvPairs map[string]string) map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	results := make(map[string]bool)
	for k, v := range kvPairs {
		s.data[k] = v
		results[k] = true
	}
	return results
}

// DeleteMultiple deletes multiple keys
// Returns number of keys deleted
func (s *Store) DeleteMultiple(keys []string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, k := range keys {
		if _, ok := s.data[k]; ok {
			delete(s.data, k)
			count++
		}
	}
	return count
}

// Count returns total number of keys in store
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}
