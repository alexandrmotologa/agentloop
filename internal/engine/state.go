package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// State holds the execution state variables with concurrency safety and deep copying.
type State struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewState creates a state object initialized with optional initial maps.
func NewState(initialMaps ...map[string]any) *State {
	s := &State{
		data: make(map[string]any),
	}
	for _, m := range initialMaps {
		for k, v := range m {
			s.data[k] = deepCopy(v)
		}
	}
	return s
}

// Get returns the value at the specified key or dot-delimited path (e.g. "extracted_data.total_amount").
func (s *State) Get(path string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path = strings.TrimPrefix(path, ".")
	path = strings.TrimPrefix(path, "state.")

	if val, ok := s.data[path]; ok {
		return deepCopy(val), true
	}

	parts := strings.Split(path, ".")
	var current any = s.data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}

	return deepCopy(current), true
}

// Set stores a value at the specified key or dot-delimited path.
func (s *State) Set(path string, val any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path = strings.TrimPrefix(path, ".")
	path = strings.TrimPrefix(path, "state.")

	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		s.data[parts[0]] = deepCopy(val)
		return
	}

	current := s.data
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, exists := current[part]
		if !exists {
			nextMap := make(map[string]any)
			current[part] = nextMap
			current = nextMap
			continue
		}
		nextMap, ok := next.(map[string]any)
		if !ok {
			nextMap = make(map[string]any)
			current[part] = nextMap
		}
		current = nextMap
	}

	current[parts[len(parts)-1]] = deepCopy(val)
}

// ApplyPatch merges a patch map into the state.
func (s *State) ApplyPatch(patch map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range patch {
		s.data[k] = deepCopy(v)
	}
}

// ToMap returns a deep copy of the underlying data map.
func (s *State) ToMap() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copyMap := make(map[string]any, len(s.data))
	for k, v := range s.data {
		copyMap[k] = deepCopy(v)
	}
	return copyMap
}

// ToJSON serializes state data to a JSON string.
func (s *State) ToJSON() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bytes, err := json.Marshal(s.data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize state to json: %w", err)
	}
	return string(bytes), nil
}

// FromJSON loads state data from a JSON string.
func (s *State) FromJSON(jsonStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return fmt.Errorf("failed to deserialize state from json: %w", err)
	}
	s.data = data
	return nil
}

// Clone creates an independent deep copy of the state.
func (s *State) Clone() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return NewState(s.data)
}

func deepCopy(val any) any {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case map[string]any:
		res := make(map[string]any, len(v))
		for k, val := range v {
			res[k] = deepCopy(val)
		}
		return res
	case []any:
		res := make([]any, len(v))
		for i, item := range v {
			res[i] = deepCopy(item)
		}
		return res
	default:
		// Primitive types or json roundtrip
		data, err := json.Marshal(v)
		if err != nil {
			return v
		}
		var out any
		if err := json.Unmarshal(data, &out); err != nil {
			return v
		}
		return out
	}
}
