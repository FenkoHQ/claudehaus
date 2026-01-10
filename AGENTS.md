# CLAUDEHAUS // AGENT INSTRUCTIONS

## Project Overview

Claudehaus is a self-hosted web application for monitoring and controlling multiple Claude Code instances via hooks. Pure Go backend, HTMX frontend, Fenko brutalist design.

## Issue Tracking

This project uses **bd (beads)** for issue tracking.
Run `bd prime` for workflow context, or install hooks (`bd hooks install`) for auto-injection.

**Quick reference:**
- `bd ready` - Find unblocked work
- `bd create "Title" --type task --priority 2` - Create issue
- `bd close <id>` - Complete work
- `bd sync` - Sync with git (run at session end)

For full workflow details: `bd prime`

---

## Technology Stack

### Backend
- **Language:** Go 1.22+
- **HTTP Router:** net/http (stdlib) or chi
- **WebSocket:** gorilla/websocket or nhooyr.io/websocket
- **Templating:** html/template (stdlib)
- **Config:** JSON file (~/.claudehaus/config.json)
- **No database** - JSON persistence + in-memory state

### Frontend
- **HTMX:** 2.0+ for dynamic HTML updates
- **Styling:** Custom CSS (Fenko brutalist design system)
- **JavaScript:** Minimal vanilla JS for WebSocket + keyboard shortcuts
- **Fonts:** JetBrains Mono (monospace primary)
- **No build step** - Static assets embedded in binary

### Development
- **Task Runner:** Make
- **Linting:** golangci-lint
- **Formatting:** gofmt / goimports
- **Testing:** go test

---

## Commands

### Build & Run
```bash
# Development
go run ./cmd/claudehaus

# Build binary
go build -o claudehaus ./cmd/claudehaus

# Build with embedded assets
go build -tags embed -o claudehaus ./cmd/claudehaus
```

### Quality Gates
```bash
# Run before committing
make check      # or: go vet ./... && golangci-lint run

# Tests
go test ./...

# Format
gofmt -w .
goimports -w .
```

### Beads Commands
```bash
bd ready              # Find available work
bd show <id>          # View issue details  
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

---

## Code Conventions

### Go
- Use stdlib where possible (net/http, html/template, encoding/json)
- Error handling: wrap errors with context (`fmt.Errorf("doing X: %w", err)`)
- Logging: use `log/slog` structured logging
- Context: pass `context.Context` as first parameter
- Naming: follow Go conventions (MixedCaps, not underscores)

### HTML Templates
- Use Go's html/template
- Partials in `web/templates/partials/`
- HTMX attributes for dynamic behavior
- No inline styles - use CSS classes

### CSS (Fenko Design System)
- Colors: Void Black (#0A0A0A), Fennec Amber (#FF9E45), Structure Gray (#333333), Signal Green (#4CDA9A), Fox Red (#D64A3A)
- Typography: Monospace (JetBrains Mono), uppercase labels
- Sharp corners, 1px borders, visible grid
- No gradients, no rounded corners, no shadows
- Instant hover inversions (no transitions except shimmer)

### JavaScript
- Vanilla JS only, no frameworks
- Single file: `web/static/js/claudehaus.js`
- Handle WebSocket connection and keyboard shortcuts
- Use data attributes for element targeting

---

## File Structure

```
claudehaus/
├── cmd/claudehaus/main.go    # Entry point
├── internal/
│   ├── server/               # HTTP server, routes, handlers
│   ├── hooks/                # Hook processing, approval logic
│   ├── session/              # Session management
│   └── config/               # Config loading, token management
├── web/
│   ├── static/css/           # Fenko styles
│   ├── static/js/            # WebSocket + shortcuts
│   └── templates/            # Go HTML templates
├── scripts/claudehaus-hook   # Companion hook script
├── go.mod
├── Makefile
└── REQ.md                    # Requirements document
```

---

## Key Design Decisions

1. **Companion hook script** - Single script users install, handles forwarding + chaining via `--chain` flag
2. **Blocking approval flow** - PermissionRequest hooks block waiting for web UI decision
3. **Passthrough timeout default** - If no response, let Claude Code terminal prompt handle it
4. **Session identity from hooks** - Use `session_id` from hook input, not generated IDs
5. **Token auth only** - No user accounts, multiple tokens allowed

---

## Testing Strategy

- Unit tests for core logic (session manager, config, approval)
- Integration tests for HTTP handlers
- Manual testing for WebSocket and UI
- No e2e browser tests in v1

---

## Landing the Plane (Session Completion)

**When ending a work session**, complete ALL steps:

1. **Run quality gates** - `make check` or `go vet ./...`
2. **Update issues** - `bd close <completed-ids>`
3. **Sync beads** - `bd sync --from-main`
4. **Commit** - `git add . && git commit -m "..."`
5. **Push** - `git push` (MANDATORY - work is NOT complete until pushed)
