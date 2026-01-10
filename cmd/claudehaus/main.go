package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/aliadnani/claudehaus/internal/config"
	"github.com/aliadnani/claudehaus/internal/server"
)

var (
	version = "dev"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		host    string
		port    int
		debug   bool
		showVer bool
	)

	flag.StringVar(&host, "host", "127.0.0.1", "Host to bind to")
	flag.IntVar(&port, "port", 8420, "Port to listen on")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.BoolVar(&showVer, "version", false, "Show version")
	flag.Parse()

	if showVer {
		fmt.Printf("claudehaus %s\n", version)
		return nil
	}

	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	token, created, err := cfg.EnsureDefaultToken()
	if err != nil {
		return fmt.Errorf("ensuring default token: %w", err)
	}
	if created {
		slog.Info("created default token", "token", token)
		fmt.Printf("\n╔════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  FIRST RUN - SAVE THIS TOKEN                               ║\n")
		fmt.Printf("╠════════════════════════════════════════════════════════════╣\n")
		fmt.Printf("║  %s  ║\n", token)
		fmt.Printf("╚════════════════════════════════════════════════════════════╝\n\n")
	}

	if host != "127.0.0.1" {
		cfg.Server.Host = host
	}
	if port != 8420 {
		cfg.Server.Port = port
	}

	srv := server.New(cfg)
	return srv.Run()
}
