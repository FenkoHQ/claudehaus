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
	// Check for subcommands first
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "tokens":
			return runTokensCommand()
		}
	}

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
		fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  CLAUDEHAUS FIRST RUN                                                        ║\n")
		fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════╣\n")
		fmt.Printf("║                                                                              ║\n")
		fmt.Printf("║  1. Save this token:                                                         ║\n")
		fmt.Printf("║     %s                     ║\n", token)
		fmt.Printf("║                                                                              ║\n")
		fmt.Printf("║  2. Set environment variables:                                               ║\n")
		fmt.Printf("║     export CLAUDEHAUS_TOKEN=\"%s\"   ║\n", token)
		fmt.Printf("║     export CLAUDEHAUS_URL=\"http://127.0.0.1:8420\"                            ║\n")
		fmt.Printf("║                                                                              ║\n")
		fmt.Printf("║  3. Add to Claude Code settings (~/.claude/settings.json):                   ║\n")
		fmt.Printf("║                                                                              ║\n")
		fmt.Printf("║     {                                                                        ║\n")
		fmt.Printf("║       \"hooks\": {                                                             ║\n")
		fmt.Printf("║         \"PreToolUse\": [{ \"matcher\": \"*\", \"hooks\": [{                        ║\n")
		fmt.Printf("║           \"type\": \"command\", \"command\": \"claudehaus-hook\" }]}],             ║\n")
		fmt.Printf("║         \"PermissionRequest\": [{ \"matcher\": \"*\", \"hooks\": [{                 ║\n")
		fmt.Printf("║           \"type\": \"command\", \"command\": \"claudehaus-hook\" }]}],             ║\n")
		fmt.Printf("║         \"PostToolUse\": [{ \"matcher\": \"*\", \"hooks\": [{                       ║\n")
		fmt.Printf("║           \"type\": \"command\", \"command\": \"claudehaus-hook\" }]}],             ║\n")
		fmt.Printf("║         \"SessionStart\": [{ \"hooks\": [{                                       ║\n")
		fmt.Printf("║           \"type\": \"command\", \"command\": \"claudehaus-hook\" }]}],             ║\n")
		fmt.Printf("║         \"SessionEnd\": [{ \"hooks\": [{                                         ║\n")
		fmt.Printf("║           \"type\": \"command\", \"command\": \"claudehaus-hook\" }]}]              ║\n")
		fmt.Printf("║       }                                                                      ║\n")
		fmt.Printf("║     }                                                                        ║\n")
		fmt.Printf("║                                                                              ║\n")
		fmt.Printf("║  4. Ensure claudehaus-hook is in your PATH:                                  ║\n")
		fmt.Printf("║     cp scripts/claudehaus-hook ~/.local/bin/                                 ║\n")
		fmt.Printf("║                                                                              ║\n")
		fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n\n")
	}

	if host != "127.0.0.1" {
		cfg.Server.Host = host
	}
	if port != 8420 {
		cfg.Server.Port = port
	}

	// Print all available tokens for web UI login
	tokens := cfg.ListTokens()
	if len(tokens) > 0 {
		fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  AUTHENTICATION TOKENS                                                      ║\n")
		fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════╣\n")
		for i, t := range tokens {
			if i > 0 {
				fmt.Printf("╠──────────────────────────────────────────────────────────────────────────────╣\n")
			}
			fmt.Printf("║  Name: %-64s ║\n", t.Name)
			if t.Value != "" {
				fmt.Printf("║  Token: %s ║\n", t.Value)
			} else {
				fmt.Printf("║  Token: [value not stored - recreate to see it]                             ║\n")
			}
		}
		fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n")
		fmt.Printf("\n  Run 'claudehaus tokens create' to create a new token\n")
		fmt.Printf("  Run 'claudehaus tokens list' to list all tokens\n\n")
	}

	srv := server.New(cfg)
	return srv.Run()
}

func runTokensCommand() error {
	// Create a new flag set for subcommands
	tokensFlag := flag.NewFlagSet("tokens", flag.ExitOnError)
	tokensFlag.Parse(os.Args[2:])

	args := tokensFlag.Args()
	if len(args) == 0 {
		return fmt.Errorf("usage: claudehaus tokens <list|create|revoke>")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	switch args[0] {
	case "list":
		return tokensList(cfg)
	case "create":
		return tokensCreate(cfg, args[1:])
	case "revoke":
		return tokensRevoke(cfg, args[1:])
	default:
		return fmt.Errorf("unknown tokens command: %s (use list, create, or revoke)", args[0])
	}
}

func tokensList(cfg *config.Config) error {
	tokens := cfg.ListTokens()
	if len(tokens) == 0 {
		fmt.Println("No tokens found.")
		return nil
	}

	fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  TOKENS                                                                      ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════╣\n")
	for i, t := range tokens {
		if i > 0 {
			fmt.Printf("╠──────────────────────────────────────────────────────────────────────────────╣\n")
		}
		fmt.Printf("║  ID:    %-64s ║\n", t.ID)
		fmt.Printf("║  Name:  %-64s ║\n", t.Name)
		if t.Value != "" {
			fmt.Printf("║  Token: %-64s ║\n", t.Value)
		} else {
			fmt.Printf("║  Token: [value not stored - recreate to see it]                             ║\n")
		}
		fmt.Printf("║  Created: %-61s ║\n", t.CreatedAt)
	}
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n\n")
	return nil
}

func tokensCreate(cfg *config.Config, args []string) error {
	name := "unnamed"
	if len(args) > 0 {
		name = args[0]
	}

	token, err := cfg.CreateToken(name)
	if err != nil {
		return fmt.Errorf("creating token: %w", err)
	}

	fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  TOKEN CREATED                                                                ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  Name:  %-64s ║\n", name)
	fmt.Printf("║  Token: %-64s ║\n", token)
	fmt.Printf("║                                                                              ║\n")
	fmt.Printf("║  Use this token to login to the web UI or set CLAUDEHAUS_TOKEN             ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n\n")
	return nil
}

func tokensRevoke(cfg *config.Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: claudehaus tokens revoke <token-id>")
	}

	tokenID := args[0]
	if !cfg.RevokeToken(tokenID) {
		return fmt.Errorf("token not found: %s", tokenID)
	}

	fmt.Printf("Token %s revoked\n", tokenID)
	return nil
}
