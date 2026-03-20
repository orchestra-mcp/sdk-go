package helpers

import (
	"encoding/json"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// ViewHint tells UI clients how to render the result.
type ViewHint string

const (
	ViewTable      ViewHint = "table"
	ViewDetailCard ViewHint = "detail_card"
	ViewKanban     ViewHint = "kanban"
	ViewDashboard  ViewHint = "dashboard"
	ViewChart      ViewHint = "chart"
	ViewTimeline   ViewHint = "timeline"
	ViewTree       ViewHint = "tree"
	ViewDiff       ViewHint = "diff"
	ViewList       ViewHint = "list"
)

// UIAction represents a contextual next-step button for UI clients.
type UIAction struct {
	Label   string         `json:"label"`
	Tool    string         `json:"tool"`
	Params  map[string]any `json:"params,omitempty"`
	Kind    string         `json:"kind,omitempty"`    // "primary", "secondary", "danger"
	Confirm bool           `json:"confirm,omitempty"` // require user confirmation before executing
}

// UIPagination holds pagination metadata for list results.
type UIPagination struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// UIColumn describes a column in table or kanban views.
type UIColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type,omitempty"` // "text", "status", "priority", "date", "badge"
}

// UIGroup describes a grouping for kanban or dashboard views.
type UIGroup struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Color string `json:"color,omitempty"`
}

// UIMetadata is the top-level structure embedded as "_ui" in tool results.
// Terminal clients never see this — the stdio translator extracts only "text".
// UI clients (WebGate) extract it and include it as "_meta" in the MCP response.
type UIMetadata struct {
	View       ViewHint      `json:"view"`
	EntityType string        `json:"entity_type,omitempty"`
	Data       any           `json:"data,omitempty"`
	Columns    []UIColumn    `json:"columns,omitempty"`
	Actions    []UIAction    `json:"actions,omitempty"`
	Pagination *UIPagination `json:"pagination,omitempty"`
	Groups     []UIGroup     `json:"groups,omitempty"`
}

// RichResult creates a ToolResponse containing both markdown text and UI metadata.
// Terminal clients see only the markdown. UI clients can read "_ui" for rich rendering.
// If ui is nil, this behaves identically to TextResult.
func RichResult(text string, ui *UIMetadata) *pluginv1.ToolResponse {
	m := map[string]any{
		"text": text,
	}
	if ui != nil {
		raw, err := json.Marshal(ui)
		if err == nil {
			var uiMap map[string]any
			if json.Unmarshal(raw, &uiMap) == nil {
				m["_ui"] = uiMap
			}
		}
	}
	s, _ := structpb.NewStruct(m)
	return &pluginv1.ToolResponse{
		Success: true,
		Result:  s,
	}
}
