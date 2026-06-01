package actorstate

import (
	"actors/internal/types"
	"sync"
	"time"
)

type State struct {
	mu          sync.RWMutex
	ID          string
	Position    types.Position
	Destination types.Position
	Path        []types.Position
	Interval    time.Duration
}

func (s *State) GetPosition() types.Position {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Position
}

func (s *State) SetPosition(pos types.Position) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Position = pos
}

func (s *State) GetID() string {
	s.mu.RLock()
	defer s.mu.Unlock()
	return s.ID
}

func (s *State) SetID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ID = id
}
