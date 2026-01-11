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
	events    *hooks.EventStore
	hub       *Hub
	templates *Templates
}

func New(cfg *config.Config) *Server {
	templates, err := NewTemplates()
	if err != nil {
		panic(err)
	}
	return &Server{
		cfg:       cfg,
		sessions:  session.NewStore(),
		approvals: hooks.NewApprovalStore(),
		events:    hooks.NewEventStore(),
		hub:       NewHub(),
		templates: templates,
	}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	tokens := s.cfg.ListTokens()
	slog.Info("loaded authentication tokens",
		"config_path", "~/.claudehaus/config.json",
		"token_count", len(tokens))

	for _, t := range tokens {
		if t.Value != "" {
			slog.Info("  token loaded", "name", t.Name, "id", t.ID)
		} else {
			slog.Warn("  token missing value (created before value storage)", "name", t.Name, "id", t.ID)
		}
	}

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
	if err := s.templates.Render(w, "base.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
