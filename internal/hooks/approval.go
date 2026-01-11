package hooks

import (
	"encoding/json"
	"sync"
	"time"
)

type ApprovalStore struct {
	mu        sync.RWMutex
	approvals map[string]*PendingApproval
}

type PendingApproval struct {
	ID           string          `json:"id"`
	SessionID    string          `json:"session_id"`
	CreatedAt    time.Time       `json:"created_at"`
	ExpiresAt    time.Time       `json:"expires_at"`
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	Prompt       string          `json:"prompt"`
	ResponseChan chan Decision   `json:"-"`
}

type Decision struct {
	Behavior string `json:"behavior"`
	Message  string `json:"message,omitempty"`
}

func NewApprovalStore() *ApprovalStore {
	return &ApprovalStore{
		approvals: make(map[string]*PendingApproval),
	}
}

func (s *ApprovalStore) Add(approval *PendingApproval) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[approval.ID] = approval
}

func (s *ApprovalStore) Get(id string) (*PendingApproval, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.approvals[id]
	return a, ok
}

func (s *ApprovalStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.approvals, id)
}

func (s *ApprovalStore) GetBySession(sessionID string) []*PendingApproval {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*PendingApproval
	for _, a := range s.approvals {
		if a.SessionID == sessionID {
			result = append(result, a)
		}
	}
	return result
}

func (s *ApprovalStore) CountBySession(sessionID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, a := range s.approvals {
		if a.SessionID == sessionID {
			count++
		}
	}
	return count
}
