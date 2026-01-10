# CLAUDEHAUS // REQUIREMENTS

```
╔═══════════════════════════════════════════════════════════════════════════╗
║  CLAUDEHAUS v1.0                                                          ║
║  MULTI-INSTANCE CLAUDE CODE CONTROL INTERFACE                             ║
║  ─────────────────────────────────────────────────────────────────────────║
║  STATUS: DESIGN PHASE                                                     ║
╚═══════════════════════════════════════════════════════════════════════════╝
```

## OVERVIEW

Claudehaus is a self-hosted web application that provides centralized monitoring and control of multiple Claude Code instances. Users can observe real-time hook events and approve/deny permission requests from a single brutalist web interface.

---

## FUNCTIONAL REQUIREMENTS

### F1: COMPANION HOOK SCRIPT

| ID | Requirement |
|----|-------------|
| F1.1 | Provide a single executable `claudehaus-hook` (shell script or Go binary) that Claude Code calls as a hook command |
| F1.2 | Script forwards all hook event data to Claudehaus server via HTTP POST |
| F1.3 | Support `--chain <command>` flag to execute additional hooks after forwarding |
| F1.4 | Chain commands receive the same stdin input as the primary hook |
| F1.5 | Chain command output is passed through (for hooks that use stdout) |
| F1.6 | Script reads server URL from `CLAUDEHAUS_URL` env var or `~/.claudehaus/config.json` |
| F1.7 | Script reads auth token from `CLAUDEHAUS_TOKEN` env var or `~/.claudehaus/config.json` |

**Example usage in Claude Code settings:**
```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "*",
      "hooks": [{
        "type": "command",
        "command": "claudehaus-hook --chain '/path/to/discord-notify.sh'"
      }]
    }]
  }
}
```

### F2: PERMISSION APPROVAL FLOW

| ID | Requirement |
|----|-------------|
| F2.1 | When `PermissionRequest` hook fires, companion script blocks and waits for server response |
| F2.2 | Web UI displays pending permission request with [ALLOW] and [DENY] buttons |
| F2.3 | User decision (allow/deny) is sent back to waiting companion script |
| F2.4 | Companion script outputs appropriate JSON to approve or deny the tool call |
| F2.5 | Configurable timeout per Claudehaus installation (not per-session) |
| F2.6 | Default timeout behavior: passthrough (return no decision, let Claude Code terminal prompt handle it) |
| F2.7 | Timeout value configurable in settings (default: 30 seconds) |

**Approval response format (per Claude Code hooks spec):**
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow"
    }
  }
}
```

### F3: REAL-TIME MONITORING

| ID | Requirement |
|----|-------------|
| F3.1 | Display live stream of hook events via WebSocket |
| F3.2 | Events appear in chronological order (newest at top) within each session |
| F3.3 | Support filtering events by type (PreToolUse, PostToolUse, etc.) |
| F3.4 | Show event details: timestamp, hook event name, tool name, tool input summary |
| F3.5 | Expandable event details for full tool_input JSON |

### F4: SESSION MANAGEMENT

| ID | Requirement |
|----|-------------|
| F4.1 | Sessions identified by `session_id` from hook input (unique identifier) |
| F4.2 | Sessions displayed by project path (`CLAUDE_PROJECT_DIR`) as default label |
| F4.3 | User can assign nicknames to sessions via web UI |
| F4.4 | Nicknames persist across Claudehaus restarts |
| F4.5 | Session list shows status indicators: active (receiving events), idle, has pending approval |
| F4.6 | Sessions with pending approvals shimmer/pulse in sidebar |
| F4.7 | `SessionStart` hook creates/registers a session |
| F4.8 | `SessionEnd` hook marks session as ended |

### F5: NOTIFICATION DISPLAY

| ID | Requirement |
|----|-------------|
| F5.1 | Notifications are displayed per-session (not global) |
| F5.2 | `Notification` hook events appear as toasts within the session panel |
| F5.3 | `idle_prompt` notifications indicate Claude is waiting for input |
| F5.4 | `permission_prompt` notifications are handled by approval UI (F2) |

### F6: AUTHENTICATION

| ID | Requirement |
|----|-------------|
| F6.1 | Token-based authentication for API and WebSocket |
| F6.2 | First token auto-generated on initial Claudehaus startup |
| F6.3 | Token stored in `~/.claudehaus/config.json` |
| F6.4 | Support multiple tokens (create, list, revoke via settings UI) |
| F6.5 | Web UI prompts for token on first visit, stores in cookie/localStorage |
| F6.6 | Tokens are auth-only, no identity meaning |
| F6.7 | API requests without valid token receive 401 Unauthorized |

### F7: KEYBOARD SHORTCUTS

| ID | Requirement |
|----|-------------|
| F7.1 | `y` or `a` - Allow pending permission request (when focused) |
| F7.2 | `n` or `d` - Deny pending permission request (when focused) |
| F7.3 | `j` / `↓` - Navigate down in lists (sessions or events) |
| F7.4 | `k` / `↑` - Navigate up in lists |
| F7.5 | `?` - Show keyboard shortcut help modal |
| F7.6 | `/` - Focus search/filter input |
| F7.7 | `Esc` - Clear focus, close modals |
| F7.8 | `1-9` - Quick-switch between first 9 sessions |
| F7.9 | `e` - Expand/collapse selected event details |
| F7.10 | `t` - Open transcript for current session (if available) |

### F8: SETTINGS & CONFIGURATION

| ID | Requirement |
|----|-------------|
| F8.1 | Settings accessible via gear icon or `/settings` route |
| F8.2 | Configure approval timeout value |
| F8.3 | Configure timeout behavior (deny / allow / passthrough) |
| F8.4 | Manage tokens (create, revoke, copy) |
| F8.5 | Manage session nicknames |
| F8.6 | Export/import configuration |

---

## NON-FUNCTIONAL REQUIREMENTS

### NF1: TECHNOLOGY STACK

| ID | Requirement |
|----|-------------|
| NF1.1 | Backend: Pure Go (no CGO dependencies) |
| NF1.2 | Frontend: Go templates + HTMX + minimal vanilla JS |
| NF1.3 | WebSocket for real-time event streaming |
| NF1.4 | Single binary deployment (templates and assets embedded) |
| NF1.5 | No external database (JSON file persistence) |
| NF1.6 | No JavaScript build step required |

### NF2: PERFORMANCE

| ID | Requirement |
|----|-------------|
| NF2.1 | Event latency < 100ms from hook fire to UI display |
| NF2.2 | Support at least 20 concurrent Claude Code sessions |
| NF2.3 | Memory usage < 100MB under normal load |
| NF2.4 | Startup time < 1 second |

### NF3: DESIGN LANGUAGE (FENKO BRUTALIST)

| ID | Requirement |
|----|-------------|
| NF3.1 | Color palette: Void Black (#0A0A0A), Fennec Amber (#FF9E45), Structure Gray (#333333), Signal Green (#4CDA9A), Fox Red (#D64A3A) |
| NF3.2 | Typography: Monospace primary (JetBrains Mono / Fira Code), uppercase labels |
| NF3.3 | Visible grid structure, 1px borders, sharp corners |
| NF3.4 | No animations except: shimmer for pending approvals, instant hover inversions |
| NF3.5 | Terminal aesthetic: cursor underscores, [brackets], >_ prompts |
| NF3.6 | High contrast, zero ambiguity |

### NF4: SECURITY

| ID | Requirement |
|----|-------------|
| NF4.1 | Bind to 127.0.0.1 by default (localhost only) |
| NF4.2 | Optional flag to bind to 0.0.0.0 for network access |
| NF4.3 | Token required for all API and WebSocket connections |
| NF4.4 | No sensitive data logged to stdout |
| NF4.5 | Tokens stored with secure file permissions (0600) |

### NF5: DEVELOPER EXPERIENCE

| ID | Requirement |
|----|-------------|
| NF5.1 | Single command to start: `claudehaus` or `claudehaus serve` |
| NF5.2 | `claudehaus init` generates config with token |
| NF5.3 | `claudehaus token create` / `token list` / `token revoke` commands |
| NF5.4 | Clear error messages with actionable guidance |
| NF5.5 | `--debug` flag for verbose logging |
| NF5.6 | Health check endpoint at `/health` |

### NF6: PERSISTENCE

| ID | Requirement |
|----|-------------|
| NF6.1 | Config stored in `~/.claudehaus/config.json` |
| NF6.2 | Persist: tokens, session nicknames, settings |
| NF6.3 | Do NOT persist: events, session history (ephemeral) |
| NF6.4 | Atomic writes to prevent corruption |

---

## ARCHITECTURE DECISIONS

### AD1: Companion Script Approach

**Decision:** Use a companion hook script rather than direct curl commands.

**Rationale:**
- Clean chaining mechanism via `--chain` flag
- Can handle blocking/waiting for approval responses
- Abstracts away API details from Claude Code config
- Can work offline/queue events in future versions

### AD2: Passthrough as Default Timeout

**Decision:** When approval timeout occurs, return "no decision" and let Claude Code's terminal prompt handle it.

**Rationale:**
- No workflow gets blocked if user doesn't respond in web UI
- User can still approve locally in terminal
- Safer than auto-allow, less frustrating than auto-deny

### AD3: Session Identity from Hook Data

**Decision:** Use `session_id` from hook input as unique identifier, not generated IDs.

**Rationale:**
- Claude Code already provides unique session IDs
- No identity synchronization needed
- Project path provides human-readable default label
- Nicknames for disambiguation are UI-only

### AD4: HTMX for Frontend

**Decision:** Use HTMX + Go templates instead of a JavaScript framework.

**Rationale:**
- Minimal client-side complexity
- No build step
- Fits brutalist aesthetic (raw HTML, no hydration)
- Fast page loads, server-rendered
- Small vanilla JS for WebSocket only

### AD5: Token Authentication

**Decision:** Shared secret tokens, not user accounts.

**Rationale:**
- Self-hosted, single-user tool
- Simple to implement and understand
- Multiple tokens allow device-specific access
- Easy to revoke/rotate

---

## HOOK EVENTS HANDLED

| Hook Event | Purpose in Claudehaus |
|------------|----------------------|
| `SessionStart` | Register new session, show in sidebar |
| `SessionEnd` | Mark session as ended |
| `PreToolUse` | Display in event feed (monitoring) |
| `PostToolUse` | Display in event feed (monitoring) |
| `PermissionRequest` | **CRITICAL**: Show approval UI, block for response |
| `Notification` | Display as per-session toast |
| `Stop` | Display in event feed, mark session idle |
| `SubagentStop` | Display in event feed |

**Not handled (v1):**
- `PreCompact` - Not user-actionable
- `UserPromptSubmit` - No remote prompt injection planned

---

## API DESIGN

### REST Endpoints

```
POST   /api/hooks/{event}     # Receive hook event from companion script
GET    /api/sessions          # List active sessions
GET    /api/sessions/{id}     # Get session details
PATCH  /api/sessions/{id}     # Update session (nickname)
POST   /api/approvals/{id}    # Submit approval decision
GET    /api/settings          # Get current settings
PATCH  /api/settings          # Update settings
POST   /api/tokens            # Create new token
GET    /api/tokens            # List tokens (masked)
DELETE /api/tokens/{id}       # Revoke token
GET    /health                # Health check (no auth)
```

### WebSocket

```
WS     /ws                    # Real-time event stream
```

**WebSocket message types:**
```json
{"type": "event", "session_id": "...", "data": {...}}
{"type": "approval_request", "session_id": "...", "approval_id": "...", "data": {...}}
{"type": "approval_resolved", "approval_id": "...", "decision": "allow|deny"}
{"type": "session_update", "session_id": "...", "status": "active|idle|ended"}
```

### Approval Blocking Flow

1. Companion script POSTs to `/api/hooks/PermissionRequest`
2. Server creates pending approval, broadcasts via WebSocket
3. Server holds HTTP connection open (long-poll)
4. User clicks allow/deny in UI → POST to `/api/approvals/{id}`
5. Server resolves pending approval, responds to held connection
6. Companion script receives response, outputs to Claude Code

---

## DATA MODELS

### Config (persisted to ~/.claudehaus/config.json)

```json
{
  "server": {
    "host": "127.0.0.1",
    "port": 8420
  },
  "tokens": [
    {
      "id": "tok_abc123",
      "name": "default",
      "value_hash": "sha256...",
      "created_at": "2025-01-11T...",
      "last_used_at": "2025-01-11T..."
    }
  ],
  "sessions": {
    "session_abc123": {
      "nickname": "main-refactor"
    }
  },
  "settings": {
    "approval_timeout_seconds": 30,
    "approval_timeout_behavior": "passthrough"
  }
}
```

### Session (in-memory)

```go
type Session struct {
    ID           string            // from hook session_id
    ProjectDir   string            // from CLAUDE_PROJECT_DIR
    Nickname     string            // user-assigned, persisted
    Status       SessionStatus     // active, idle, ended
    StartedAt    time.Time
    LastEventAt  time.Time
    Events       []HookEvent       // ring buffer, last N events
    Pending      []PendingApproval
}
```

### HookEvent (in-memory)

```go
type HookEvent struct {
    ID            string
    SessionID     string
    Timestamp     time.Time
    EventName     string          // PreToolUse, PostToolUse, etc.
    ToolName      string          // if applicable
    ToolInput     json.RawMessage
    ToolResponse  json.RawMessage // PostToolUse only
}
```

### PendingApproval (in-memory)

```go
type PendingApproval struct {
    ID           string
    SessionID    string
    CreatedAt    time.Time
    ExpiresAt    time.Time
    ToolName     string
    ToolInput    json.RawMessage
    ResponseChan chan ApprovalDecision // blocks companion script
}
```

---

## UI LAYOUT

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ ░▒▓ CLAUDEHAUS ▓▒░                                    [?] HELP  [⚙] CONFIG │
├─────────────────┬───────────────────────────────────────────────────────────┤
│                 │                                                           │
│ SESSIONS        │  SESSION: [ my-project ]  ● ACTIVE                        │
│ ─────────────── │  PATH: /home/user/code/my-project                         │
│                 │  ───────────────────────────────────────────────────────  │
│ ● my-project    │                                                           │
│   api-backend   │  ┌─ PENDING APPROVAL ─────────────────────────────────┐   │
│ ◌ frontend      │  │                                                     │  │
│                 │  │  TOOL: Bash                                         │  │
│                 │  │  > rm -rf node_modules && npm install               │  │
│                 │  │                                                     │  │
│                 │  │  [Y] ALLOW                         [N] DENY         │  │
│                 │  │                                                     │  │
│                 │  │  TIMEOUT: 28s remaining                             │  │
│                 │  └─────────────────────────────────────────────────────┘  │
│                 │                                                           │
│                 │  ─────────────────────────────────────────────────────── │
│                 │  EVENT FEED                                               │
│                 │  ─────────────────────────────────────────────────────── │
│                 │  12:03:04 │ PostToolUse │ Write  │ src/main.go           │
│                 │  12:03:02 │ PreToolUse  │ Read   │ go.mod                │
│                 │  12:03:01 │ SessionStart│        │                       │
│                 │                                                           │
├─────────────────┴───────────────────────────────────────────────────────────┤
│ > READY // PRESS ? FOR SHORTCUTS                                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## FILE STRUCTURE

```
claudehaus/
├── cmd/
│   └── claudehaus/
│       └── main.go           # CLI entry point
├── internal/
│   ├── server/
│   │   ├── server.go         # HTTP server setup
│   │   ├── routes.go         # Route definitions
│   │   ├── handlers.go       # HTTP handlers
│   │   └── websocket.go      # WebSocket hub
│   ├── hooks/
│   │   ├── handler.go        # Hook event processing
│   │   └── approval.go       # Approval blocking logic
│   ├── session/
│   │   ├── manager.go        # Session lifecycle
│   │   └── store.go          # In-memory session store
│   ├── config/
│   │   ├── config.go         # Config loading/saving
│   │   └── tokens.go         # Token management
│   └── ui/
│       └── templates/        # Go templates
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   └── fenko.css     # Brutalist styles
│   │   └── js/
│   │       └── claudehaus.js # WebSocket + shortcuts
│   └── templates/
│       ├── base.html
│       ├── index.html
│       ├── session.html
│       └── settings.html
├── scripts/
│   └── claudehaus-hook       # Companion hook script
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## COMPANION SCRIPT (claudehaus-hook)

```bash
#!/usr/bin/env bash
# claudehaus-hook - Forward Claude Code hooks to Claudehaus server
#
# Usage: claudehaus-hook [--chain <command>]
#
# Environment:
#   CLAUDEHAUS_URL    Server URL (default: http://127.0.0.1:8420)
#   CLAUDEHAUS_TOKEN  Auth token
#
# Reads hook input from stdin, forwards to server, optionally chains.

set -euo pipefail

# ... implementation
```

**Key behaviors:**
1. Read stdin into variable (hook input JSON)
2. Extract `hook_event_name` from JSON
3. POST to `$CLAUDEHAUS_URL/api/hooks/$hook_event_name`
4. For `PermissionRequest`: wait for response, output JSON
5. For other hooks: fire-and-forget
6. If `--chain` specified: pipe stdin to chain command, pass through output

---

## OPEN QUESTIONS (v2+)

- [ ] Transcript viewer integration (read from `transcript_path`)
- [ ] Event history persistence (optional SQLite)
- [ ] Multi-user support with user accounts
- [ ] Remote access with proper TLS
- [ ] Mobile-friendly responsive layout
- [ ] Browser notifications for pending approvals
- [ ] Webhook integrations (Slack, Discord) as first-class feature

---

```
╔═══════════════════════════════════════════════════════════════════════════╗
║  END OF REQUIREMENTS // AWAITING APPROVAL TO PROCEED                      ║
╚═══════════════════════════════════════════════════════════════════════════╝
```
