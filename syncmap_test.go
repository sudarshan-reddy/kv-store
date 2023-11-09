package kv

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	inMemoryMap := NewWriteOptimizedMapStore(0, false)
	var tests = []struct {
		key   string
		value string
	}{
		{"foo", "bar"},
		{"bar", "baz"},
		{"baz", "qux"},
		{"qux", "quux"},
		{"quux", "corge"},
		{"corge", "grault"},
	}
	for _, test := range tests {
		inMemoryMap.Put(test.key, test.value)
		value, err := inMemoryMap.Get(test.key)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if value.(string) != test.value {
			t.Errorf("Expected value '%v', got %v", test.value, value.(string))
		}
	}

}

func TestConcurrentPuts(t *testing.T) {
	m := NewWriteOptimizedMapStore(1, false)
	m.db = make(map[string]interface{})

	// Use a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Define the total number of concurrent operations
	totalOperations := 100

	// Keep track of what we expect to see in the map
	expectedMap := make(map[string]interface{}, totalOperations)

	// Launch several goroutines for concurrent Put operations
	for i := 0; i < totalOperations; i++ {
		wg.Add(1)
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		expectedMap[key] = value
		go func(k string, v interface{}) {
			defer wg.Done()
			err := m.Put(k, v)
			if err != nil {
				t.Errorf("Put returned an error: %v", err)
			}
		}(key, value)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Lock the map to make a safe copy of its current state for comparison
	m.m.Lock()
	gotMap := make(map[string]interface{}, len(m.db))
	for k, v := range m.db {
		gotMap[k] = v
	}
	m.m.Unlock()

	// Compare the expected map with the actual map
	if !reflect.DeepEqual(expectedMap, gotMap) {
		t.Errorf("Concurrent Puts resulted in incorrect map state.\nExpected: %v\nGot: %v", expectedMap, gotMap)
	}
}

func TestBatchUpdate(t *testing.T) {
	tests := []struct {
		name             string
		initialState     map[string]interface{}
		pairs            []Pair
		rollbackEnabled  bool
		wantErr          bool
		wantState        map[string]interface{}
		wantUpdatedPairs []Pair
	}{
		{
			name:             "successful batch update",
			initialState:     map[string]interface{}{"a": 1, "b": 2},
			pairs:            []Pair{{"a", 2}, {"b", 3}},
			rollbackEnabled:  true,
			wantErr:          false,
			wantState:        map[string]interface{}{"a": 2, "b": 3},
			wantUpdatedPairs: []Pair{{"a", 2}, {"b", 3}},
		},
		{
			name:         "rollback on context cancellation",
			initialState: map[string]interface{}{"a": 1, "b": 2},
			// This is a very dubious test. I set a high number of pairs to simulate work being done
			// and time taken. This is the number that works on my machine as time taken to do work for
			// greater than 10 nanoseconds.
			// TODO: If I have time, give the map a Clock implementation in which I can deliberately
			// introduce a delay for tests.
			pairs:            aLotOfPairs(10000),
			rollbackEnabled:  true,
			wantErr:          true,
			wantState:        map[string]interface{}{"a": 1, "b": 2}, // State should revert to initial
			wantUpdatedPairs: nil,
		},
		{
			name:             "ignore non-existent keys",
			initialState:     map[string]interface{}{"a": 1},
			pairs:            []Pair{{"a", 2}, {"nonExistentKey", 3}},
			rollbackEnabled:  false,
			wantErr:          false,
			wantState:        map[string]interface{}{"a": 2},
			wantUpdatedPairs: []Pair{{"a", 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			m := NewWriteOptimizedMapStore(1, tt.rollbackEnabled)
			m.db = tt.initialState

			var updatedPairs []Pair
			var err error

			if tt.rollbackEnabled {
				// Simulate cancellation during the operation
				go func() {
					time.Sleep(10 * time.Nanosecond) // Simulate work being done before cancellation
					cancel()
				}()
			}

			updatedPairs, err = m.BatchUpdate(ctx, tt.pairs)

			if !tt.rollbackEnabled && err != nil {
				t.Errorf("BatchUpdate() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(updatedPairs, tt.wantUpdatedPairs) {
				t.Errorf("BatchUpdate() got updatedPairs = %v, want %v", updatedPairs, tt.wantUpdatedPairs)
			}

			if !reflect.DeepEqual(m.db, tt.wantState) {
				t.Errorf("BatchUpdate() got state = %v, want %v", m.db, tt.wantState)
			}

			// Cleanup
			cancel()
		})
	}
}

func aLotOfPairs(n int) []Pair {
	pairs := make([]Pair, 0)
	for i := 0; i < n; i++ {
		pairs = append(pairs, Pair{strconv.Itoa(i), i})
	}

	return pairs
}
