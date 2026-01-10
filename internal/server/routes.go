package server

import (
	"net/http"
)

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)

	mux.HandleFunc("GET /", s.handleIndex)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("GET /partials/sessions", s.handlePartialSessions)
	mux.HandleFunc("GET /partials/session/{id}", s.handlePartialSessionDetail)

	api := http.NewServeMux()
	api.HandleFunc("POST /hooks/{event}", s.handleHook)
	api.HandleFunc("GET /sessions", s.handleListSessions)
	api.HandleFunc("GET /sessions/{id}", s.handleGetSession)
	api.HandleFunc("PATCH /sessions/{id}", s.handleUpdateSession)
	api.HandleFunc("POST /approvals/{id}", s.handleApproval)
	api.HandleFunc("GET /settings", s.handleGetSettings)
	api.HandleFunc("PATCH /settings", s.handleUpdateSettings)
	api.HandleFunc("POST /tokens", s.handleCreateToken)
	api.HandleFunc("GET /tokens", s.handleListTokens)
	api.HandleFunc("DELETE /tokens/{id}", s.handleRevokeToken)

	mux.Handle("/api/", http.StripPrefix("/api", s.authMiddleware(api)))

	mux.HandleFunc("GET /ws", s.handleWebSocket)
}
