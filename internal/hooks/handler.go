package hooks

import (
	"encoding/json"
)

type HookInput struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	Cwd            string          `json:"cwd"`
	PermissionMode string          `json:"permission_mode"`
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name,omitempty"`
	ToolInput      json.RawMessage `json:"tool_input,omitempty"`
	ToolResponse   json.RawMessage `json:"tool_response,omitempty"`
	ToolUseID      string          `json:"tool_use_id,omitempty"`
	Message        string          `json:"message,omitempty"`
	NotificationType string        `json:"notification_type,omitempty"`
	Prompt         string          `json:"prompt,omitempty"`
	StopHookActive bool            `json:"stop_hook_active,omitempty"`
	Reason         string          `json:"reason,omitempty"`
	Source         string          `json:"source,omitempty"`
}

type ApprovalDecision struct {
	Behavior string `json:"behavior"`
}

type ApprovalResponse struct {
	HookSpecificOutput struct {
		HookEventName string           `json:"hookEventName"`
		Decision      ApprovalDecision `json:"decision"`
	} `json:"hookSpecificOutput"`
}

func NewAllowResponse() ApprovalResponse {
	resp := ApprovalResponse{}
	resp.HookSpecificOutput.HookEventName = "PermissionRequest"
	resp.HookSpecificOutput.Decision.Behavior = "allow"
	return resp
}

func NewDenyResponse(message string) ApprovalResponse {
	resp := ApprovalResponse{}
	resp.HookSpecificOutput.HookEventName = "PermissionRequest"
	resp.HookSpecificOutput.Decision.Behavior = "deny"
	return resp
}

type HookEvent struct {
	ID           string          `json:"id"`
	SessionID    string          `json:"session_id"`
	Timestamp    string          `json:"timestamp"`
	EventName    string          `json:"event_name"`
	ToolName     string          `json:"tool_name,omitempty"`
	ToolInput    json.RawMessage `json:"tool_input,omitempty"`
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`
}
