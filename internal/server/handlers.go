package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/aliadnani/claudehaus/internal/hooks"
	"github.com/aliadnani/claudehaus/internal/session"
)

func (s *Server) handleHook(w http.ResponseWriter, r *http.Request) {
	event := r.PathValue("event")

	var input hooks.HookInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	sess, exists := s.sessions.Get(input.SessionID)
	if !exists {
		sess = &session.Session{
			ID:          input.SessionID,
			ProjectDir:  input.Cwd,
			Nickname:    filepath.Base(input.Cwd),
			Status:      session.StatusActive,
			StartedAt:   time.Now(),
			LastEventAt: time.Now(),
		}
		if meta, ok := s.cfg.Sessions[input.SessionID]; ok {
			sess.Nickname = meta.Nickname
		}
		s.sessions.Set(sess)
	}

	sess.LastEventAt = time.Now()
	sess.Status = session.StatusActive

	s.hub.Broadcast(Message{
		Type:      "event",
		SessionID: input.SessionID,
		Data: map[string]any{
			"event_name": event,
			"tool_name":  input.ToolName,
			"timestamp":  time.Now().Format("15:04:05"),
		},
	})

	switch event {
	case "SessionStart":
		s.hub.Broadcast(Message{Type: "session_update", SessionID: input.SessionID, Data: map[string]any{"status": "active"}})
		w.WriteHeader(http.StatusOK)

	case "SessionEnd":
		s.sessions.UpdateStatus(input.SessionID, session.StatusEnded)
		s.hub.Broadcast(Message{Type: "session_update", SessionID: input.SessionID, Data: map[string]any{"status": "ended"}})
		w.WriteHeader(http.StatusOK)

	case "PermissionRequest":
		approvalID := generateID()
		timeout := time.Duration(s.cfg.Settings.ApprovalTimeoutSeconds) * time.Second

		pending := &hooks.PendingApproval{
			ID:           approvalID,
			SessionID:    input.SessionID,
			CreatedAt:    time.Now(),
			ExpiresAt:    time.Now().Add(timeout),
			ToolName:     input.ToolName,
			ToolInput:    input.ToolInput,
			ResponseChan: make(chan hooks.Decision, 1),
		}

		s.approvals.Add(pending)
		sess.HasPending = true
		sess.PendingCount = s.approvals.CountBySession(input.SessionID)

		s.hub.Broadcast(Message{
			Type:      "approval_request",
			SessionID: input.SessionID,
			Data: map[string]any{
				"approval_id": approvalID,
				"tool_name":   input.ToolName,
				"tool_input":  input.ToolInput,
				"expires_at":  pending.ExpiresAt,
			},
		})

		select {
		case decision := <-pending.ResponseChan:
			s.approvals.Remove(approvalID)
			sess.PendingCount = s.approvals.CountBySession(input.SessionID)
			sess.HasPending = sess.PendingCount > 0

			var resp hooks.ApprovalResponse
			if decision.Behavior == "allow" {
				resp = hooks.NewAllowResponse()
			} else {
				resp = hooks.NewDenyResponse(decision.Message)
			}
			writeJSON(w, resp)

		case <-time.After(timeout):
			s.approvals.Remove(approvalID)
			sess.PendingCount = s.approvals.CountBySession(input.SessionID)
			sess.HasPending = sess.PendingCount > 0

			s.hub.Broadcast(Message{Type: "approval_resolved", Data: map[string]any{"approval_id": approvalID, "decision": "timeout"}})

			switch s.cfg.Settings.ApprovalTimeoutBehavior {
			case "allow":
				writeJSON(w, hooks.NewAllowResponse())
			case "deny":
				writeJSON(w, hooks.NewDenyResponse("Approval timed out"))
			default:
				w.WriteHeader(http.StatusOK)
			}
		}

	case "Stop", "SubagentStop":
		s.sessions.UpdateStatus(input.SessionID, session.StatusIdle)
		s.hub.Broadcast(Message{Type: "session_update", SessionID: input.SessionID, Data: map[string]any{"status": "idle"}})
		w.WriteHeader(http.StatusOK)

	default:
		w.WriteHeader(http.StatusOK)
	}
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := s.sessions.All()
	writeJSON(w, sessions)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.sessions.Get(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	writeJSON(w, sess)
}

func (s *Server) handleUpdateSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	sess, ok := s.sessions.Get(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	sess.Nickname = req.Nickname
	s.sessions.Set(sess)

	s.cfg.Sessions[id] = struct {
		Nickname string `json:"nickname"`
	}{Nickname: req.Nickname}
	_ = s.cfg.Save()

	writeJSON(w, sess)
}

func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Decision string `json:"decision"`
		Message  string `json:"message,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	pending, ok := s.approvals.Get(id)
	if !ok {
		http.Error(w, "approval not found", http.StatusNotFound)
		return
	}

	decision := hooks.Decision{
		Behavior: req.Decision,
		Message:  req.Message,
	}

	select {
	case pending.ResponseChan <- decision:
		s.hub.Broadcast(Message{
			Type: "approval_resolved",
			Data: map[string]any{
				"approval_id": id,
				"decision":    req.Decision,
			},
		})
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "approval already resolved", http.StatusConflict)
	}
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.cfg.Settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings struct {
		ApprovalTimeoutSeconds  *int    `json:"approval_timeout_seconds,omitempty"`
		ApprovalTimeoutBehavior *string `json:"approval_timeout_behavior,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if settings.ApprovalTimeoutSeconds != nil {
		s.cfg.Settings.ApprovalTimeoutSeconds = *settings.ApprovalTimeoutSeconds
	}
	if settings.ApprovalTimeoutBehavior != nil {
		s.cfg.Settings.ApprovalTimeoutBehavior = *settings.ApprovalTimeoutBehavior
	}

	if err := s.cfg.Save(); err != nil {
		http.Error(w, "failed to save settings", http.StatusInternalServerError)
		return
	}

	writeJSON(w, s.cfg.Settings)
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "unnamed"
	}

	token, err := s.cfg.CreateToken(req.Name)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"token": token})
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.cfg.ListTokens())
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.cfg.RevokeToken(id) {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}



func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
