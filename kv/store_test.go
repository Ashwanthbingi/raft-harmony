package kv

import (
	"encoding/json"
	"testing"
)

// TestStoreSetGet tests basic Set/Get operations
func TestStoreSetGet(t *testing.T) {
	store := NewStore()

	// Test 1: Set and Get
	store.Set("key1", "value1")
	val, ok := store.Get("key1")

	if !ok {
		t.Errorf("Expected key1 to exist")
	}

	if val != "value1" {
		t.Errorf("Expected 'value1', got '%s'", val)
	}

	// Test 2: Get non-existent key
	_, ok = store.Get("nonexistent")
	if ok {
		t.Errorf("Expected nonexistent key to return false")
	}

	// Test 3: Overwrite value
	store.Set("key1", "new_value")
	val, ok = store.Get("key1")

	if val != "new_value" {
		t.Errorf("Expected 'new_value', got '%s'", val)
	}
}

// TestStoreConcurrency tests thread-safe operations
func TestStoreConcurrency(t *testing.T) {
	store := NewStore()
	done := make(chan bool)

	// Write goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.Set("key", "value")
		}
		done <- true
	}()

	// Read goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.Get("key")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

// TestStoreSerialize tests JSON serialization for snapshots
func TestStoreSerialize(t *testing.T) {
	store := NewStore()

	// Add test data
	store.Set("x", "10")
	store.Set("y", "20")
	store.Set("z", "30")

	// Serialize
	data := store.Serialize()

	// Verify it's valid JSON
	var restored map[string]string
	err := json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Serialize produced invalid JSON: %v", err)
	}

	// Verify contents
	if restored["x"] != "10" {
		t.Errorf("Expected x=10, got %s", restored["x"])
	}
	if restored["y"] != "20" {
		t.Errorf("Expected y=20, got %s", restored["y"])
	}
	if restored["z"] != "30" {
		t.Errorf("Expected z=30, got %s", restored["z"])
	}
}

// TestStoreRestore tests restoring from snapshot
func TestStoreRestore(t *testing.T) {
	// Create original store
	original := NewStore()
	original.Set("a", "1")
	original.Set("b", "2")
	original.Set("c", "3")

	// Serialize
	data := original.Serialize()

	// Create new store and restore
	restored := NewStore()
	err := restored.Restore(data)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify restored state
	tests := []struct {
		key   string
		value string
	}{
		{"a", "1"},
		{"b", "2"},
		{"c", "3"},
	}

	for _, test := range tests {
		val, ok := restored.Get(test.key)
		if !ok {
			t.Errorf("Key %s not found after restore", test.key)
		}
		if val != test.value {
			t.Errorf("Key %s: Expected %s, got %s", test.key, test.value, val)
		}
	}
}

// TestStoreRestoreEmpty tests restoring empty state
func TestStoreRestoreEmpty(t *testing.T) {
	store := NewStore()
	empty := []byte("{}")

	err := store.Restore(empty)
	if err != nil {
		t.Fatalf("Restore empty JSON failed: %v", err)
	}

	_, ok := store.Get("any_key")
	if ok {
		t.Errorf("Expected empty store after restore")
	}
}

// TestStoreSnapshotRoundTrip tests snapshot generation and restoration
func TestStoreSnapshotRoundTrip(t *testing.T) {
	// Original store
	s1 := NewStore()
	s1.Set("db", "postgres")
	s1.Set("host", "localhost")
	s1.Set("port", "5432")

	// Take snapshot
	snapshot := s1.Serialize()

	// New store restores from snapshot
	s2 := NewStore()
	s2.Restore(snapshot)

	// Verify all keys preserved
	val, ok := s2.Get("db")
	if !ok || val != "postgres" {
		t.Errorf("db key not restored correctly")
	}

	val, ok = s2.Get("host")
	if !ok || val != "localhost" {
		t.Errorf("host key not restored correctly")
	}

	val, ok = s2.Get("port")
	if !ok || val != "5432" {
		t.Errorf("port key not restored correctly")
	}
}

// TestStoreLargeSnapshot tests large state serialization
func TestStoreLargeSnapshot(t *testing.T) {
	store := NewStore()

	// Add 1000 key-value pairs
	for i := 0; i < 1000; i++ {
		key := "key_" + string(rune(i%10000))
		val := "value_" + string(rune(i%10000))
		store.Set(key, val)
	}

	// Serialize
	data := store.Serialize()

	// Restore in new store
	newStore := NewStore()
	err := newStore.Restore(data)
	if err != nil {
		t.Fatalf("Failed to restore large snapshot: %v", err)
	}

	// Verify some keys
	val, ok := newStore.Get("key_0")
	if !ok {
		t.Errorf("Large snapshot restore failed: key_0 not found")
	}
	if val == "" {
		t.Errorf("Large snapshot restore failed: value is empty")
	}
}
