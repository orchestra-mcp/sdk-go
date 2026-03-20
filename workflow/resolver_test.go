package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orchestra-mcp/sdk-go/globaldb"
)

func setupResolverDB(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".orchestra", "db"), 0700)
	t.Setenv("HOME", tmpDir)
	globaldb.Close()
	t.Cleanup(func() { globaldb.Close() })
}

func TestResolver_AutoSeedsDefaultWorkflow(t *testing.T) {
	setupResolverDB(t)
	fallback := DefaultEngine()
	resolver := NewResolver(fallback)

	// Resolve a project — should auto-seed the default workflow in DB.
	eng := resolver.Resolve("auto-seed-project")

	// Engine should behave like the default workflow (todo -> in-progress).
	nextFromTodo := eng.NextStates(StateID("todo"))
	if len(nextFromTodo) == 0 {
		t.Fatal("expected transitions from 'todo' in auto-seeded engine")
	}
	if nextFromTodo[0] != StateID("in-progress") {
		t.Errorf("expected first transition from todo to be in-progress, got %s", nextFromTodo[0])
	}

	// Verify DB record was created.
	rec, err := globaldb.GetProjectWorkflow("auto-seed-project")
	if err != nil {
		t.Fatalf("expected DB record after auto-seed, got error: %v", err)
	}
	if rec.Name != "orchestra-default" {
		t.Errorf("expected workflow name 'orchestra-default', got %q", rec.Name)
	}
}

func TestResolver_ReturnsDBWorkflow(t *testing.T) {
	setupResolverDB(t)
	fallback := DefaultEngine()
	resolver := NewResolver(fallback)

	// Create a custom workflow in DB with different states.
	rec := &globaldb.WorkflowRecord{
		ID:           "WFL-TST",
		ProjectID:    "test-project",
		Name:         "Custom Workflow",
		InitialState: "open",
		IsDefault:    true,
		States: map[string]globaldb.WorkflowStateRec{
			"open":   {Label: "Open", Terminal: false, ActiveWork: true},
			"closed": {Label: "Closed", Terminal: true, ActiveWork: false},
		},
		Transitions: []globaldb.WorkflowTransitionRec{
			{From: "open", To: "closed"},
		},
		Gates: map[string]globaldb.WorkflowGateRec{},
	}
	if err := globaldb.CreateWorkflowRecord(rec); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	eng := resolver.Resolve("test-project")

	// Verify the custom engine has the right states.
	if !eng.IsTerminal(StateID("closed")) {
		t.Error("expected 'closed' to be terminal in custom workflow")
	}
	if eng.IsTerminal(StateID("open")) {
		t.Error("expected 'open' to not be terminal in custom workflow")
	}
	// Verify default states are NOT present in the custom workflow.
	nextFromTodo := eng.NextStates(StateID("todo"))
	if len(nextFromTodo) > 0 {
		t.Error("expected no transitions from 'todo' in custom workflow")
	}
}

func TestResolver_CachesResult(t *testing.T) {
	setupResolverDB(t)
	fallback := DefaultEngine()
	resolver := NewResolver(fallback)

	// First call (auto-seeds).
	eng1 := resolver.Resolve("cached-project")
	// Second call should hit cache.
	eng2 := resolver.Resolve("cached-project")

	if eng1 != eng2 {
		t.Error("expected same engine pointer from cache")
	}
}

func TestResolver_InvalidateForcesFetch(t *testing.T) {
	setupResolverDB(t)
	fallback := DefaultEngine()
	resolver := NewResolver(fallback)

	// First resolve auto-seeds the default workflow.
	eng1 := resolver.Resolve("test-project")
	// Should have default workflow behavior.
	if !eng1.CanTransition(StateID("todo"), StateID("in-progress")) {
		t.Error("expected todo -> in-progress in auto-seeded engine")
	}

	// Now update the DB workflow to a custom one.
	// First delete the auto-seeded one, then create custom.
	recs, _ := globaldb.ListWorkflowRecords("test-project")
	for _, r := range recs {
		globaldb.DeleteWorkflowRecord(r.ID)
	}
	customRec := &globaldb.WorkflowRecord{
		ID:           "WFL-CUS",
		ProjectID:    "test-project",
		Name:         "Custom After Invalidate",
		InitialState: "pending",
		IsDefault:    true,
		States: map[string]globaldb.WorkflowStateRec{
			"pending":  {Label: "Pending", Terminal: false},
			"complete": {Label: "Complete", Terminal: true},
		},
		Transitions: []globaldb.WorkflowTransitionRec{
			{From: "pending", To: "complete"},
		},
		Gates: map[string]globaldb.WorkflowGateRec{},
	}
	if err := globaldb.CreateWorkflowRecord(customRec); err != nil {
		t.Fatalf("create custom workflow: %v", err)
	}

	// Without invalidation, cache still returns the old engine.
	eng2 := resolver.Resolve("test-project")
	if eng2 != eng1 {
		t.Error("expected cached engine before invalidation")
	}

	// After invalidation, should pick up the custom DB record.
	resolver.Invalidate("test-project")
	eng3 := resolver.Resolve("test-project")
	if !eng3.IsTerminal(StateID("complete")) {
		t.Error("expected 'complete' terminal in custom engine after invalidation")
	}
	if eng3.CanTransition(StateID("todo"), StateID("in-progress")) {
		t.Error("expected no todo -> in-progress in custom engine")
	}
}

func TestResolver_InvalidateAll(t *testing.T) {
	setupResolverDB(t)
	fallback := DefaultEngine()
	resolver := NewResolver(fallback)

	// Populate cache.
	resolver.Resolve("project-a")
	resolver.Resolve("project-b")

	// Clear all.
	resolver.InvalidateAll()

	// Check cache is empty.
	resolver.mu.RLock()
	cacheLen := len(resolver.cache)
	resolver.mu.RUnlock()
	if cacheLen != 0 {
		t.Errorf("expected empty cache after InvalidateAll, got %d entries", cacheLen)
	}
}

func TestResolver_FallbackAccessor(t *testing.T) {
	fallback := DefaultEngine()
	resolver := NewResolver(fallback)

	if resolver.Fallback() != fallback {
		t.Error("Fallback() should return the fallback engine")
	}
}

func TestResolver_EmptyProjectIDReturnsFallback(t *testing.T) {
	fallback := DefaultEngine()
	resolver := NewResolver(fallback)

	eng := resolver.Resolve("")
	if eng != fallback {
		t.Error("expected fallback for empty projectID")
	}
}

func TestRecordToDefinition(t *testing.T) {
	rec := &globaldb.WorkflowRecord{
		Name:         "Test WF",
		Description:  "Desc",
		InitialState: "start",
		States: map[string]globaldb.WorkflowStateRec{
			"start": {Label: "Start", Terminal: false, ActiveWork: true},
			"end":   {Label: "End", Terminal: true, ActiveWork: false},
		},
		Transitions: []globaldb.WorkflowTransitionRec{
			{From: "start", To: "end", Gate: "review"},
		},
		Gates: map[string]globaldb.WorkflowGateRec{
			"review": {
				Label:           "Review",
				RequiredSection: "Results",
				FilePatterns:    []string{"*.test.go"},
				SkippableFor:    []string{"hotfix"},
			},
		},
	}

	def := recordToDefinition(rec)

	if def.Name != "Test WF" {
		t.Errorf("Name = %q, want %q", def.Name, "Test WF")
	}
	if def.InitialState != StateID("start") {
		t.Errorf("InitialState = %q, want %q", def.InitialState, "start")
	}
	if len(def.States) != 2 {
		t.Errorf("States count = %d, want 2", len(def.States))
	}
	if !def.States[StateID("end")].Terminal {
		t.Error("expected 'end' to be terminal")
	}
	if len(def.Transitions) != 1 {
		t.Errorf("Transitions count = %d, want 1", len(def.Transitions))
	}
	if def.Transitions[0].Gate != "review" {
		t.Errorf("Transition gate = %q, want %q", def.Transitions[0].Gate, "review")
	}
	if len(def.Gates) != 1 {
		t.Errorf("Gates count = %d, want 1", len(def.Gates))
	}
	gate := def.Gates["review"]
	if gate.RequiredSection != "Results" {
		t.Errorf("Gate RequiredSection = %q, want %q", gate.RequiredSection, "Results")
	}
	if len(gate.SkippableFor) != 1 || gate.SkippableFor[0] != "hotfix" {
		t.Errorf("Gate SkippableFor = %v, want [hotfix]", gate.SkippableFor)
	}
}

func TestSeedDefaultWorkflow(t *testing.T) {
	setupResolverDB(t)

	// First seed should create a record.
	rec, err := SeedDefaultWorkflow("seed-test-project")
	if err != nil {
		t.Fatalf("SeedDefaultWorkflow: %v", err)
	}
	if rec == nil {
		t.Fatal("expected a workflow record from first seed")
	}
	if rec.Name != "orchestra-default" {
		t.Errorf("expected name 'orchestra-default', got %q", rec.Name)
	}
	if !rec.IsDefault {
		t.Error("expected IsDefault to be true")
	}
	if len(rec.States) != 7 {
		t.Errorf("expected 7 states, got %d", len(rec.States))
	}

	// Second seed should be a no-op.
	rec2, err := SeedDefaultWorkflow("seed-test-project")
	if err != nil {
		t.Fatalf("second SeedDefaultWorkflow: %v", err)
	}
	if rec2 != nil {
		t.Error("expected nil from second seed (already exists)")
	}
}
