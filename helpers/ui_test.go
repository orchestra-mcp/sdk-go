package helpers

import (
	"testing"

	"github.com/orchestra-mcp/sdk-go/types"
)

func TestRichResult_IncludesTextAndUI(t *testing.T) {
	ui := &UIMetadata{View: ViewTable, EntityType: "feature"}
	resp := RichResult("# Features\n\n...", ui)

	if !resp.Success {
		t.Fatal("expected Success=true")
	}

	// Text field preserved.
	textVal := resp.Result.GetFields()["text"].GetStringValue()
	if textVal != "# Features\n\n..." {
		t.Errorf("text: got %q, want %q", textVal, "# Features\n\n...")
	}

	// _ui field present.
	uiVal := resp.Result.GetFields()["_ui"]
	if uiVal == nil {
		t.Fatal("expected _ui field in result")
	}
	uiStruct := uiVal.GetStructValue()
	if uiStruct == nil {
		t.Fatal("expected _ui to be a struct")
	}
	view := uiStruct.GetFields()["view"].GetStringValue()
	if view != "table" {
		t.Errorf("view: got %q, want %q", view, "table")
	}
	entType := uiStruct.GetFields()["entity_type"].GetStringValue()
	if entType != "feature" {
		t.Errorf("entity_type: got %q, want %q", entType, "feature")
	}
}

func TestRichResult_NilUI_SameAsTextResult(t *testing.T) {
	rich := RichResult("hello", nil)
	text := TextResult("hello")

	// Both should have the same text field.
	richText := rich.Result.GetFields()["text"].GetStringValue()
	textText := text.Result.GetFields()["text"].GetStringValue()
	if richText != textText {
		t.Errorf("text mismatch: rich=%q text=%q", richText, textText)
	}

	// RichResult with nil UI should NOT have _ui field.
	if _, hasUI := rich.Result.GetFields()["_ui"]; hasUI {
		t.Error("expected no _ui field when ui is nil")
	}
}

func TestRichResult_WithActions(t *testing.T) {
	ui := &UIMetadata{
		View:       ViewDetailCard,
		EntityType: "feature",
		Actions: []UIAction{
			{Label: "Advance", Tool: "advance_feature", Kind: "primary"},
		},
	}
	resp := RichResult("feature detail", ui)

	uiStruct := resp.Result.GetFields()["_ui"].GetStructValue()
	actions := uiStruct.GetFields()["actions"].GetListValue()
	if actions == nil || len(actions.GetValues()) != 1 {
		t.Fatalf("expected 1 action, got %v", actions)
	}
	action := actions.GetValues()[0].GetStructValue()
	label := action.GetFields()["label"].GetStringValue()
	if label != "Advance" {
		t.Errorf("action label: got %q, want %q", label, "Advance")
	}
}

func TestRichResult_WithPagination(t *testing.T) {
	ui := &UIMetadata{
		View:       ViewTable,
		EntityType: "feature",
		Pagination: &UIPagination{Total: 42, Limit: 50, Offset: 0},
	}
	resp := RichResult("table", ui)

	uiStruct := resp.Result.GetFields()["_ui"].GetStructValue()
	pg := uiStruct.GetFields()["pagination"].GetStructValue()
	if pg == nil {
		t.Fatal("expected pagination in _ui")
	}
	total := pg.GetFields()["total"].GetNumberValue()
	if total != 42 {
		t.Errorf("pagination.total: got %v, want 42", total)
	}
}

func TestFeatureListUI_Columns(t *testing.T) {
	features := []*types.FeatureData{
		{ID: "FEAT-ABC", Title: "Test", Status: "todo", Priority: "P1"},
	}
	pg := PaginationParams{Limit: 50, Offset: 0}
	ui := FeatureListUI(features, pg, 1)

	if ui.View != ViewTable {
		t.Errorf("view: got %q, want %q", ui.View, ViewTable)
	}
	if ui.EntityType != "feature" {
		t.Errorf("entity_type: got %q, want %q", ui.EntityType, "feature")
	}
	if len(ui.Columns) != 6 {
		t.Errorf("columns: got %d, want 6", len(ui.Columns))
	}
	if ui.Pagination.Total != 1 {
		t.Errorf("pagination.total: got %d, want 1", ui.Pagination.Total)
	}
}

func TestFeatureDetailUI_WithActions(t *testing.T) {
	f := &types.FeatureData{ID: "FEAT-ABC", Title: "Test"}
	ui := FeatureDetailUI(f,
		UIAction{Label: "Start", Tool: "set_current_feature", Kind: "primary"},
		UIAction{Label: "Delete", Tool: "delete_feature", Kind: "danger", Confirm: true},
	)

	if ui.View != ViewDetailCard {
		t.Errorf("view: got %q, want %q", ui.View, ViewDetailCard)
	}
	if len(ui.Actions) != 2 {
		t.Errorf("actions: got %d, want 2", len(ui.Actions))
	}
	if ui.Actions[1].Confirm != true {
		t.Error("expected second action to have Confirm=true")
	}
}

func TestProgressUI(t *testing.T) {
	counts := map[string]int{"todo": 5, "done": 3}
	ui := ProgressUI(counts, 8, 3, 37.5)

	if ui.View != ViewDashboard {
		t.Errorf("view: got %q, want %q", ui.View, ViewDashboard)
	}
	data := ui.Data.(map[string]any)
	if data["total"] != 8 {
		t.Errorf("total: got %v, want 8", data["total"])
	}
}

func TestWorkflowKanbanUI(t *testing.T) {
	features := []*types.FeatureData{
		{ID: "FEAT-A", Status: "todo"},
		{ID: "FEAT-B", Status: "in-progress"},
	}
	counts := map[string]int{"todo": 1, "in-progress": 1}
	ui := WorkflowKanbanUI(features, counts, 2)

	if ui.View != ViewKanban {
		t.Errorf("view: got %q, want %q", ui.View, ViewKanban)
	}
	if len(ui.Groups) != 7 {
		t.Errorf("groups: got %d, want 7", len(ui.Groups))
	}
}

func TestPlanListUI(t *testing.T) {
	plans := []*types.PlanData{
		{ID: "PLAN-ABC", Title: "Test Plan", Status: "draft"},
	}
	pg := PaginationParams{Limit: 50, Offset: 0}
	ui := PlanListUI(plans, pg, 1)

	if ui.View != ViewTable {
		t.Errorf("view: got %q, want %q", ui.View, ViewTable)
	}
	if len(ui.Columns) != 4 {
		t.Errorf("columns: got %d, want 4", len(ui.Columns))
	}
}
