package session

import (
	"sync"
	"time"
)

type Status string

const (
	StatusActive Status = "active"
	StatusIdle   Status = "idle"
	StatusEnded  Status = "ended"
)

type Session struct {
	ID             string    `json:"id"`
	ProjectDir     string    `json:"project_dir"`
	Nickname       string    `json:"nickname"`
	Status         Status    `json:"status"`
	StartedAt      time.Time `json:"started_at"`
	LastEventAt    time.Time `json:"last_event_at"`
	HasPending     bool      `json:"has_pending"`
	PendingCount   int       `json:"pending_count"`
}

type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[string]*Session),
	}
}

func (s *Store) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

func (s *Store) Set(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
}

func (s *Store) All() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		result = append(result, sess)
	}
	return result
}

func (s *Store) UpdateStatus(id string, status Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		sess.Status = status
		sess.LastEventAt = time.Now()
	}
}
