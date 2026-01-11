package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliadnani/claudehaus/internal/hooks"
	"github.com/aliadnani/claudehaus/internal/session"
)

func (s *Server) handleHook(w http.ResponseWriter, r *http.Request) {
	event := r.PathValue("event")

	var input hooks.HookInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.Warn("invalid hook request body", "error", err, "event", event)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slog.Info("hook event received",
		"event", event,
		"session_id", input.SessionID,
		"tool_name", input.ToolName,
		"cwd", input.Cwd)

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
		slog.Info("new session created",
			"session_id", input.SessionID,
			"nickname", sess.Nickname,
			"project_dir", sess.ProjectDir)
	} else {
		slog.Debug("existing session found", "session_id", input.SessionID, "nickname", sess.Nickname)
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
		s.events.AddEvent(input.SessionID, "SessionStart", "", "", "Session started")
		s.hub.Broadcast(Message{Type: "session_update", SessionID: input.SessionID, Data: map[string]any{"status": "active"}})
		slog.Info("session started", "session_id", input.SessionID, "nickname", sess.Nickname)
		w.WriteHeader(http.StatusOK)

	case "SessionEnd":
		s.events.AddEvent(input.SessionID, "SessionEnd", "", "", "Session ended")
		s.sessions.UpdateStatus(input.SessionID, session.StatusEnded)
		s.hub.Broadcast(Message{Type: "session_update", SessionID: input.SessionID, Data: map[string]any{"status": "ended"}})
		slog.Info("session ended", "session_id", input.SessionID, "nickname", sess.Nickname)
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
			Prompt:       input.Prompt,
			ResponseChan: make(chan hooks.Decision, 1),
		}

		s.approvals.Add(pending)
		sess.HasPending = true
		sess.PendingCount = s.approvals.CountBySession(input.SessionID)

		slog.Info("permission request pending",
			"approval_id", approvalID,
			"session_id", input.SessionID,
			"tool_name", input.ToolName,
			"timeout_seconds", s.cfg.Settings.ApprovalTimeoutSeconds)

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

			slog.Info("permission request resolved",
				"approval_id", approvalID,
				"decision", decision.Behavior,
				"message", decision.Message)

			s.events.AddEvent(input.SessionID, "PermissionRequest", input.ToolName, string(input.ToolInput),
				"Approved: "+decision.Behavior)

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

			slog.Warn("permission request timed out",
				"approval_id", approvalID,
				"timeout_behavior", s.cfg.Settings.ApprovalTimeoutBehavior)

			s.events.AddEvent(input.SessionID, "PermissionRequest", input.ToolName, string(input.ToolInput),
				"Timed out")

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
		s.events.AddEvent(input.SessionID, event, "", "", "Task stopped")
		s.sessions.UpdateStatus(input.SessionID, session.StatusIdle)
		s.hub.Broadcast(Message{Type: "session_update", SessionID: input.SessionID, Data: map[string]any{"status": "idle"}})
		slog.Info("session idle", "session_id", input.SessionID, "event", event)
		w.WriteHeader(http.StatusOK)

	default:
		// Capture all other events (PreToolUse, PostToolUse, etc.) for the web UI
		if input.ToolName != "" {
			s.events.AddEvent(input.SessionID, event, input.ToolName, string(input.ToolInput), "")
		} else {
			s.events.AddEvent(input.SessionID, event, "", "", "")
		}
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

	var decision, message string

	// HTMX sends hx-vals as JSON when using curly brace syntax
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var req struct {
			Decision string `json:"decision"`
			Message  string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			slog.Warn("invalid approval request", "error", err, "approval_id", id)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		decision = req.Decision
		message = req.Message
	} else {
		// Fallback to form data
		if err := r.ParseForm(); err != nil {
			slog.Warn("invalid approval request", "error", err, "approval_id", id)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		decision = r.FormValue("decision")
		message = r.FormValue("message")
	}

	if decision == "" {
		slog.Warn("missing decision", "approval_id", id)
		http.Error(w, "missing decision", http.StatusBadRequest)
		return
	}

	pending, ok := s.approvals.Get(id)
	if !ok {
		slog.Warn("approval not found", "approval_id", id)
		http.Error(w, "approval not found", http.StatusNotFound)
		return
	}

	decisionStruct := hooks.Decision{
		Behavior: decision,
		Message:  message,
	}

	select {
	case pending.ResponseChan <- decisionStruct:
		slog.Info("approval decision sent via API",
			"approval_id", id,
			"decision", decision,
			"message", message)
		s.hub.Broadcast(Message{
			Type: "approval_resolved",
			Data: map[string]any{
				"approval_id": id,
				"decision":    decision,
			},
		})
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		slog.Warn("approval already resolved", "approval_id", id)
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



func (s *Server) handleVerifyToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Token == "" || !s.cfg.ValidateToken(req.Token) {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
