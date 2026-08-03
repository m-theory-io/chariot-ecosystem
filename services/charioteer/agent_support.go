package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// AgentFixture stores a reusable one-shot plan execution payload.
type AgentFixture struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Plan      string         `json:"plan"`
	AgentName string         `json:"agentName,omitempty"`
	VarsMap   map[string]any `json:"varsMap,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// AgentFixtureStore persists Agent tab fixtures on disk.
type AgentFixtureStore struct {
	mu           sync.RWMutex
	fixturesPath string
	fixtures     []AgentFixture
}

func newAgentFixtureStore(dir string) (*AgentFixtureStore, error) {
	if dir == "" {
		dir = defaultMCPDataDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create agent fixture data dir: %w", err)
	}
	store := &AgentFixtureStore{fixturesPath: filepath.Join(dir, "agent_fixtures.json")}
	if err := store.loadFixtures(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *AgentFixtureStore) loadFixtures() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.fixturesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.fixtures = nil
			return nil
		}
		return fmt.Errorf("read agent fixtures: %w", err)
	}
	if err := json.Unmarshal(data, &s.fixtures); err != nil {
		return fmt.Errorf("parse agent fixtures: %w", err)
	}
	s.sortLocked()
	return nil
}

func (s *AgentFixtureStore) ListFixtures() []AgentFixture {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AgentFixture, len(s.fixtures))
	copy(out, s.fixtures)
	return out
}

func (s *AgentFixtureStore) SaveFixture(input AgentFixture) (AgentFixture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	input.Name = strings.TrimSpace(input.Name)
	input.Plan = strings.TrimSpace(input.Plan)
	input.AgentName = strings.TrimSpace(input.AgentName)
	if input.Name == "" {
		return AgentFixture{}, errors.New("fixture name is required")
	}
	if input.Plan == "" {
		return AgentFixture{}, errors.New("plan is required")
	}
	if input.VarsMap == nil {
		input.VarsMap = map[string]any{}
	}
	if input.ID == "" {
		input.ID = generateFixtureID()
		input.CreatedAt = now
	} else {
		for i := range s.fixtures {
			if s.fixtures[i].ID == input.ID {
				input.CreatedAt = s.fixtures[i].CreatedAt
				break
			}
		}
		if input.CreatedAt.IsZero() {
			input.CreatedAt = now
		}
	}
	input.UpdatedAt = now

	replaced := false
	for i := range s.fixtures {
		if s.fixtures[i].ID == input.ID {
			s.fixtures[i] = input
			replaced = true
			break
		}
	}
	if !replaced {
		s.fixtures = append(s.fixtures, input)
	}
	s.sortLocked()
	if err := writeJSONFile(s.fixturesPath, s.fixtures); err != nil {
		return AgentFixture{}, err
	}
	return input, nil
}

func (s *AgentFixtureStore) DeleteFixture(id string) error {
	if id == "" {
		return errors.New("fixture id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.fixtures {
		if s.fixtures[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.New("fixture not found")
	}
	s.fixtures = append(s.fixtures[:idx], s.fixtures[idx+1:]...)
	return writeJSONFile(s.fixturesPath, s.fixtures)
}

func (s *AgentFixtureStore) sortLocked() {
	sort.SliceStable(s.fixtures, func(i, j int) bool {
		return s.fixtures[i].UpdatedAt.After(s.fixtures[j].UpdatedAt)
	})
}

func agentFixturesHandler(w http.ResponseWriter, r *http.Request) {
	if agentFixtureStore == nil {
		sendError(w, http.StatusInternalServerError, "Agent fixture store not available")
		return
	}
	switch r.Method {
	case http.MethodGet:
		sendSuccess(w, agentFixtureStore.ListFixtures())
	case http.MethodPost:
		var payload struct {
			Fixture AgentFixture `json:"fixture"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			sendError(w, http.StatusBadRequest, "invalid fixture payload: "+err.Error())
			return
		}
		saved, err := agentFixtureStore.SaveFixture(payload.Fixture)
		if err != nil {
			sendError(w, http.StatusBadRequest, err.Error())
			return
		}
		sendSuccess(w, saved)
	default:
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func agentFixtureItemHandler(w http.ResponseWriter, r *http.Request) {
	if agentFixtureStore == nil {
		sendError(w, http.StatusInternalServerError, "Agent fixture store not available")
		return
	}
	if r.Method != http.MethodDelete {
		sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := extractResourceID(r.URL.Path)
	if err := agentFixtureStore.DeleteFixture(id); err != nil {
		sendError(w, http.StatusBadRequest, err.Error())
		return
	}
	sendSuccess(w, map[string]string{"deleted": id})
}
