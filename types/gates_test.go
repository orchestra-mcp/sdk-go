package types_test

import (
	"strings"
	"testing"

	"github.com/orchestra-mcp/sdk-go/types"
)

func TestGetGateGatedTransition(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	if gate == nil {
		t.Fatal("expected gate for in-progress -> ready-for-testing")
	}
	if gate.ID != types.GateImplementation {
		t.Errorf("expected GateImplementation, got %s", gate.ID)
	}
	if gate.Name != "Implementation Complete" {
		t.Errorf("expected 'Implementation Complete', got %s", gate.Name)
	}
}

func TestGetGateFreeTransition(t *testing.T) {
	gate := types.GetGate(types.StatusBacklog, types.StatusTodo)
	if gate != nil {
		t.Fatal("expected no gate for backlog -> todo")
	}
}

func TestIsGated(t *testing.T) {
	tests := []struct {
		from   types.FeatureStatus
		to     types.FeatureStatus
		gated  bool
	}{
		{types.StatusInProgress, types.StatusReadyForTesting, true},
		{types.StatusInTesting, types.StatusReadyForDocs, true},
		{types.StatusInDocs, types.StatusDocumented, true},
		{types.StatusBacklog, types.StatusTodo, false},
		{types.StatusTodo, types.StatusInProgress, false},
		{types.StatusReadyForTesting, types.StatusInTesting, false},
		{types.StatusReadyForDocs, types.StatusInDocs, false},
		{types.StatusDocumented, types.StatusInReview, false},
		{types.StatusNeedsEdits, types.StatusInProgress, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			got := types.IsGated(tt.from, tt.to)
			if got != tt.gated {
				t.Errorf("IsGated(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.gated)
			}
		})
	}
}

func TestGateValidateEmptyEvidence(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	err := gate.Validate("")
	if err == nil {
		t.Fatal("expected error for empty evidence")
	}
	if !strings.Contains(err.Error(), "requires evidence") {
		t.Errorf("expected 'requires evidence' in error, got: %s", err.Error())
	}
}

func TestGateValidateTooShort(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	err := gate.Validate("done")
	if err == nil {
		t.Fatal("expected error for too-short evidence")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("expected 'too short' in error, got: %s", err.Error())
	}
}

func TestGateValidateMissingSections(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	// Long enough (>100 chars) but no sections.
	err := gate.Validate("This is a long enough string but has no markdown section headers at all. It needs to be over one hundred characters to pass the total length check first.")
	if err == nil {
		t.Fatal("expected error for missing sections")
	}
	if !strings.Contains(err.Error(), "missing required sections") {
		t.Errorf("expected 'missing required sections' in error, got: %s", err.Error())
	}
}

func TestGateValidatePartialSections(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	// Has Summary but missing Changes and Verification. Must be >100 chars total.
	err := gate.Validate("## Summary\nImplemented the login flow with full OAuth2 support including token refresh, PKCE verification, and comprehensive error handling.\n")
	if err == nil {
		t.Fatal("expected error for partial sections")
	}
	if !strings.Contains(err.Error(), "Changes") {
		t.Errorf("expected missing 'Changes' in error, got: %s", err.Error())
	}
}

func TestGateValidateEmptySectionContent(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	// Must be >100 chars total to pass MinTotalLen. Changes section is empty.
	err := gate.Validate("## Summary\nImplemented the full login flow with OAuth2 and JWT tokens.\n\n## Changes\n\n\n## Verification\nRun the entire test suite to verify all endpoints work correctly.")
	if err == nil {
		t.Fatal("expected error for empty section content")
	}
	if !strings.Contains(err.Error(), "insufficient content") {
		t.Errorf("expected 'insufficient content' in error, got: %s", err.Error())
	}
}

func TestGateValidatePassesWithValidEvidence(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	evidence := "## Summary\nImplemented the OAuth2 login flow with JWT tokens.\n\n## Changes\n- auth/handler.go (added login endpoint)\n- auth/service.go (token generation)\n\n## Verification\nCall POST /api/auth/login with valid credentials."
	err := gate.Validate(evidence)
	if err != nil {
		t.Fatalf("expected valid evidence to pass, got: %v", err)
	}
}

func TestGateValidateCaseInsensitiveSections(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	evidence := "## summary\nImplemented the OAuth2 login flow with JWT tokens.\n\n## changes\n- auth/handler.go (added login endpoint)\n\n## verification\nCall POST /api/auth/login with valid credentials."
	err := gate.Validate(evidence)
	if err != nil {
		t.Fatalf("expected case-insensitive sections to pass, got: %v", err)
	}
}

func TestGate2TestingComplete(t *testing.T) {
	gate := types.GetGate(types.StatusInTesting, types.StatusReadyForDocs)
	if gate == nil {
		t.Fatal("expected gate for in-testing -> ready-for-docs")
	}
	if gate.ID != types.GateTesting {
		t.Errorf("expected GateTesting, got %s", gate.ID)
	}

	// Valid evidence.
	err := gate.Validate("## Summary\nTested all API endpoints and edge cases.\n\n## Results\nAll 15 test cases passed without failures.\n\n## Coverage\n87% line coverage across the auth module.")
	if err != nil {
		t.Fatalf("expected valid testing evidence to pass, got: %v", err)
	}

	// Missing Results section. Must be >100 chars total.
	err = gate.Validate("## Summary\nTested all API endpoints and edge cases for the authentication module.\n\n## Coverage\nFull line and branch coverage of the auth module including error paths.")
	if err == nil {
		t.Fatal("expected error for missing Results section")
	}
}

func TestGate3DocumentationComplete(t *testing.T) {
	gate := types.GetGate(types.StatusInDocs, types.StatusDocumented)
	if gate == nil {
		t.Fatal("expected gate for in-docs -> documented")
	}
	if gate.ID != types.GateDocumentation {
		t.Errorf("expected GateDocumentation, got %s", gate.ID)
	}

	// Valid evidence.
	err := gate.Validate("## Summary\nDocumented all auth endpoints and setup guide.\n\n## Location\ndocs/api/auth.md and README.md setup section.")
	if err != nil {
		t.Fatalf("expected valid doc evidence to pass, got: %v", err)
	}

	// Missing Location. Must be >80 chars total for Gate 3.
	err = gate.Validate("## Summary\nDocumented the entire auth system thoroughly including all endpoints, configuration, and setup instructions.")
	if err == nil {
		t.Fatal("expected error for missing Location section")
	}
}

func TestReviewGate(t *testing.T) {
	gate := types.ReviewGate
	if gate == nil {
		t.Fatal("expected ReviewGate to be defined")
	}
	if gate.ID != types.GateReviewSelfCheck {
		t.Errorf("expected GateReviewSelfCheck, got %s", gate.ID)
	}

	// Valid self-review.
	err := gate.Validate("## Summary\nOAuth2 login feature with JWT tokens.\n\n## Quality\nCode follows project conventions, no known issues.\n\n## Checklist\n- [x] auth/handler.go — login endpoint implemented\n- [x] auth/handler_test.go — tests written and passing")
	if err != nil {
		t.Fatalf("expected valid self-review to pass, got: %v", err)
	}

	// Missing Quality. Must be >120 chars total for ReviewGate.
	err = gate.Validate("## Summary\nOAuth2 login feature with JWT tokens and refresh flow.\n\n## Checklist\n- [x] auth/handler.go - Login endpoint implemented and tested with comprehensive edge cases")
	if err == nil {
		t.Fatal("expected error for missing Quality section")
	}
}

func TestGateValidateRejectsEvidenceWithoutFilePaths(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusReadyForTesting)
	// Evidence with all sections but no file paths in Changes.
	evidence := "## Summary\nImplemented the login flow with full support.\n\n## Changes\nAdded the login endpoint and token generation logic\n\n## Verification\nCall the login endpoint with valid credentials."
	err := gate.Validate(evidence)
	if err == nil {
		t.Fatal("expected error for Changes section without file paths")
	}
	if !strings.Contains(err.Error(), "file path") {
		t.Errorf("expected 'file path' in error, got: %s", err.Error())
	}
}

func TestGate3RejectsLocationWithoutFilePaths(t *testing.T) {
	gate := types.GetGate(types.StatusInDocs, types.StatusDocumented)
	evidence := "## Summary\nDocumented the auth system thoroughly.\n\n## Location\nThe documentation is in the docs folder and the README."
	err := gate.Validate(evidence)
	if err == nil {
		t.Fatal("expected error for Location section without file paths")
	}
	if !strings.Contains(err.Error(), "file path") {
		t.Errorf("expected 'file path' in error, got: %s", err.Error())
	}
}

func TestAllGatesHaveChecklistsAndSections(t *testing.T) {
	for from, toMap := range types.GateRequirements {
		for to, gate := range toMap {
			if gate.Checklist == "" {
				t.Errorf("gate %s -> %s has empty checklist", from, to)
			}
			if len(gate.RequiredSections) == 0 {
				t.Errorf("gate %s -> %s has no required sections", from, to)
			}
			if gate.Name == "" {
				t.Errorf("gate %s -> %s has empty name", from, to)
			}
		}
	}

	// Also check ReviewGate.
	if types.ReviewGate.Checklist == "" {
		t.Error("ReviewGate has empty checklist")
	}
	if len(types.ReviewGate.RequiredSections) == 0 {
		t.Error("ReviewGate has no required sections")
	}
}
