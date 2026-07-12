package kv

import (
	"encoding/json"
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
