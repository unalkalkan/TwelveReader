package features

import (
	"encoding/json"
	"net/http"
	"sync"
)

// Flag represents a single feature flag with its configuration
type Flag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// Store manages feature flags in memory
type Store struct {
	flags map[string]*Flag
	mu    sync.RWMutex
}

// NewStore creates a new feature flag store with optional initial flags
func NewStore(initialFlags map[string]bool) *Store {
	s := &Store{
		flags: make(map[string]*Flag),
	}
	for name, enabled := range initialFlags {
		s.flags[name] = &Flag{
			Name:    name,
			Enabled: enabled,
		}
	}
	return s
}

// Set enables or disables a feature flag
func (s *Store) Set(name string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if flag, ok := s.flags[name]; ok {
		flag.Enabled = enabled
	} else {
		s.flags[name] = &Flag{
			Name:    name,
			Enabled: enabled,
		}
	}
}

// Enabled checks if a feature flag is enabled
func (s *Store) Enabled(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flag, ok := s.flags[name]
	if !ok {
		return false
	}
	return flag.Enabled
}

// GetAll returns all feature flags as a map (safe for JSON serialization)
func (s *Store) GetAll() map[string]Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]Flag, len(s.flags))
	for name, flag := range s.flags {
		result[name] = Flag{
			Name:        flag.Name,
			Description: flag.Description,
			Enabled:     flag.Enabled,
		}
	}
	return result
}

// Response represents the /api/v1/features response
type Response struct {
	Features map[string]Flag `json:"features"`
}

// HTTPHandler returns an HTTP handler for GET /api/v1/features
func (s *Store) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "METHOD_NOT_ALLOWED",
					"message": "Method not allowed",
				},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Features: s.GetAll(),
		})
	}
}
