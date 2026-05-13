package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/ltaoo/velo/dir"
)

// WindowState holds the saved position and size for a window.
type WindowState struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Data is the top-level structure persisted to data.json.
type Data struct {
	Windows map[string]*WindowState    `json:"windows"`
	Config  map[string]json.RawMessage `json:"config"`
}

// Store provides read/write access to data.json.
type Store struct {
	path string
	mu   sync.Mutex
	data *Data
}

// New creates a Store that reads/writes data.json beside the executable.
func New() *Store {
	p := filepath.Join(dir.ExeDir(), "data.json")
	s := &Store{
		path: p,
		data: &Data{
			Windows: make(map[string]*WindowState),
			Config:  make(map[string]json.RawMessage),
		},
	}
	s.load()
	// Ensure the file exists on disk from the start.
	s.save()
	return s
}

// Path returns the file path of data.json.
func (s *Store) Path() string {
	return s.path
}

func (s *Store) load() {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var d Data
	if err := json.Unmarshal(raw, &d); err != nil {
		return
	}
	if d.Windows == nil {
		d.Windows = make(map[string]*WindowState)
	}
	if d.Config == nil {
		d.Config = make(map[string]json.RawMessage)
	}
	s.data = &d
}

func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

// GetWindow returns the saved state for the named window, or nil if none.
func (s *Store) GetWindow(name string) *WindowState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Windows[name]
}

// SaveWindow persists position and size for the named window.
func (s *Store) SaveWindow(name string, state *WindowState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Windows[name] = state
	return s.save()
}

// Get returns the raw JSON value for the given config key, or nil if not found.
func (s *Store) Get(key string) json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Config[key]
}

// GetAll returns all config entries as a map.
func (s *Store) GetAll() map[string]json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make(map[string]json.RawMessage, len(s.data.Config))
	for k, v := range s.data.Config {
		cp[k] = v
	}
	return cp
}

// Set stores a value under the given config key and persists to disk.
func (s *Store) Set(key string, value json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Config[key] = value
	return s.save()
}

// Delete removes a config key and persists to disk.
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Config, key)
	return s.save()
}
