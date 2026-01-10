package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aliadnani/claudehaus/internal/config"
	"github.com/aliadnani/claudehaus/internal/hooks"
	"github.com/aliadnani/claudehaus/internal/session"
)

type Server struct {
	cfg       *config.Config
	sessions  *session.Store
	approvals *hooks.ApprovalStore
	hub       *Hub
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg:       cfg,
		sessions:  session.NewStore(),
		approvals: hooks.NewApprovalStore(),
		hub:       NewHub(),
	}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	slog.Info("starting server", "addr", addr)

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>CLAUDEHAUS</title></head>
<body style="background:#0A0A0A;color:#FF9E45;font-family:monospace;">
<h1>░▒▓ CLAUDEHAUS ▓▒░</h1>
<p>STATUS: OPERATIONAL</p>
</body>
</html>`))
}
