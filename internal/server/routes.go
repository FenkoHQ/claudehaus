package server

import (
	"net/http"
)

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("GET /partials/sessions", s.handlePartialSessions)
	mux.HandleFunc("GET /partials/session/{id}", s.handlePartialSessionDetail)

	mux.HandleFunc("POST /api/hooks/{event}", s.authAPIMiddleware(s.handleHook))
	mux.HandleFunc("GET /api/sessions", s.authAPIMiddleware(s.handleListSessions))
	mux.HandleFunc("GET /api/sessions/{id}", s.authAPIMiddleware(s.handleGetSession))
	mux.HandleFunc("PATCH /api/sessions/{id}", s.authAPIMiddleware(s.handleUpdateSession))
	mux.HandleFunc("POST /api/approvals/{id}", s.authAPIMiddleware(s.handleApproval))
	mux.HandleFunc("GET /api/settings", s.authAPIMiddleware(s.handleGetSettings))
	mux.HandleFunc("PATCH /api/settings", s.authAPIMiddleware(s.handleUpdateSettings))
	mux.HandleFunc("POST /api/tokens", s.authAPIMiddleware(s.handleCreateToken))
	mux.HandleFunc("GET /api/tokens", s.authAPIMiddleware(s.handleListTokens))
	mux.HandleFunc("DELETE /api/tokens/{id}", s.authAPIMiddleware(s.handleRevokeToken))

	mux.HandleFunc("GET /ws", s.handleWebSocket)

	mux.HandleFunc("GET /", s.handleIndex)
}
