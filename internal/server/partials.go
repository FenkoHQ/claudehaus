package server

import (
	"html/template"
	"net/http"
	"time"
)

var partialTemplates *template.Template

func init() {
	var err error
	partialTemplates, err = template.ParseGlob("web/templates/partials/*.html")
	if err != nil {
		partialTemplates = template.New("")
	}
}

func (s *Server) handlePartialSessions(w http.ResponseWriter, r *http.Request) {
	sessions := s.sessions.All()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partialTemplates.ExecuteTemplate(w, "sessions", sessions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type sessionDetailData struct {
	Session   any
	Approvals []approvalData
	Events    []eventData
}

type approvalData struct {
	ID            string
	ToolName      string
	ToolInput     string
	Prompt        string
	TimeRemaining int
}

type eventData struct {
	Timestamp string
	EventName string
	ToolName  string
	Detail    string
	ToolInput string
}

func (s *Server) handlePartialSessionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sess, ok := s.sessions.Get(id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	pendingApprovals := s.approvals.GetBySession(id)
	approvals := make([]approvalData, 0, len(pendingApprovals))
	for _, p := range pendingApprovals {
		remaining := int(time.Until(p.ExpiresAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		approvals = append(approvals, approvalData{
			ID:            p.ID,
			ToolName:      p.ToolName,
			ToolInput:     string(p.ToolInput),
			Prompt:        p.Prompt,
			TimeRemaining: remaining,
		})
	}

	events := s.events.GetBySession(id, 50)
	eventList := make([]eventData, 0, len(events))
	for _, e := range events {
		eventList = append(eventList, eventData{
			Timestamp: e.Timestamp,
			EventName: e.EventName,
			ToolName:  e.ToolName,
			Detail:    e.Detail,
			ToolInput: e.ToolInput,
		})
	}

	data := sessionDetailData{
		Session:   sess,
		Approvals: approvals,
		Events:    eventList,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := partialTemplates.ExecuteTemplate(w, "session_detail", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
