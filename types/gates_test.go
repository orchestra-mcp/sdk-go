package types_test

import (
	"strings"
	"testing"

	"github.com/orchestra-mcp/sdk-go/types"
)

// ---------------------------------------------------------------------------
// 1. GetGate — gated transitions
// ---------------------------------------------------------------------------

func TestGetGateCodeComplete(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	if gate == nil {
		t.Fatal("expected gate for in-progress -> in-testing")
	}
	if gate.ID != types.GateCodeComplete {
		t.Errorf("expected GateCodeComplete, got %s", gate.ID)
	}
	if gate.Name != "Code Complete" {
		t.Errorf("expected 'Code Complete', got %s", gate.Name)
	}
	if gate.RequiredSection != "Changes" {
		t.Errorf("expected RequiredSection 'Changes', got %s", gate.RequiredSection)
	}
}

func TestGetGateTestComplete(t *testing.T) {
	gate := types.GetGate(types.StatusInTesting, types.StatusInDocs)
	if gate == nil {
		t.Fatal("expected gate for in-testing -> in-docs")
	}
	if gate.ID != types.GateTestComplete {
		t.Errorf("expected GateTestComplete, got %s", gate.ID)
	}
	if gate.Name != "Test Complete" {
		t.Errorf("expected 'Test Complete', got %s", gate.Name)
	}
	if gate.RequiredSection != "Results" {
		t.Errorf("expected RequiredSection 'Results', got %s", gate.RequiredSection)
	}
}

func TestGetGateTestCompleteSkipDocs(t *testing.T) {
	gate := types.GetGate(types.StatusInTesting, types.StatusInReview)
	if gate == nil {
		t.Fatal("expected gate for in-testing -> in-review")
	}
	if gate.ID != types.GateTestComplete {
		t.Errorf("expected GateTestComplete, got %s", gate.ID)
	}
	if gate.Name != "Test Complete (skip docs)" {
		t.Errorf("expected 'Test Complete (skip docs)', got %s", gate.Name)
	}
}

func TestGetGateDocsComplete(t *testing.T) {
	gate := types.GetGate(types.StatusInDocs, types.StatusInReview)
	if gate == nil {
		t.Fatal("expected gate for in-docs -> in-review")
	}
	if gate.ID != types.GateDocsComplete {
		t.Errorf("expected GateDocsComplete, got %s", gate.ID)
	}
	if gate.Name != "Docs Complete" {
		t.Errorf("expected 'Docs Complete', got %s", gate.Name)
	}
	if gate.DocsFolder != "docs" {
		t.Errorf("expected DocsFolder 'docs', got %s", gate.DocsFolder)
	}
}

// ---------------------------------------------------------------------------
// 2. GetGate — free transitions return nil
// ---------------------------------------------------------------------------

func TestGetGateFreeTransition(t *testing.T) {
	free := []struct {
		from types.FeatureStatus
		to   types.FeatureStatus
	}{
		{types.StatusTodo, types.StatusInProgress},
		{types.StatusNeedsEdits, types.StatusInProgress},
	}
	for _, tt := range free {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			gate := types.GetGate(tt.from, tt.to)
			if gate != nil {
				t.Errorf("expected no gate for %s -> %s, got %+v", tt.from, tt.to, gate)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. IsGated — exhaustive check of all transitions
// ---------------------------------------------------------------------------

func TestIsGated(t *testing.T) {
	tests := []struct {
		from  types.FeatureStatus
		to    types.FeatureStatus
		gated bool
	}{
		// Gated transitions.
		{types.StatusInProgress, types.StatusInTesting, true},
		{types.StatusInTesting, types.StatusInDocs, true},
		{types.StatusInTesting, types.StatusInReview, true},
		{types.StatusInDocs, types.StatusInReview, true},

		// Free transitions.
		{types.StatusTodo, types.StatusInProgress, false},
		{types.StatusNeedsEdits, types.StatusInProgress, false},

		// Non-existent transitions.
		{types.StatusDone, types.StatusTodo, false},
		{types.StatusInReview, types.StatusDone, false},
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

// ---------------------------------------------------------------------------
// 4. Validate — empty evidence
// ---------------------------------------------------------------------------

func TestGateValidateEmptyEvidence(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	err := gate.Validate("")
	if err == nil {
		t.Fatal("expected error for empty evidence")
	}
	if !strings.Contains(err.Error(), "requires evidence") {
		t.Errorf("expected 'requires evidence' in error, got: %s", err.Error())
	}
}

func TestGateValidateWhitespaceOnlyEvidence(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	err := gate.Validate("   \n\n  \t  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only evidence")
	}
	if !strings.Contains(err.Error(), "requires evidence") {
		t.Errorf("expected 'requires evidence' in error, got: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// 5. Validate — too-short section content
// ---------------------------------------------------------------------------

func TestGateValidateTooShortSectionContent(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	// Has the required section but content is under 10 characters.
	err := gate.Validate("## Changes\nshort")
	if err == nil {
		t.Fatal("expected error for too-short section content")
	}
	if !strings.Contains(err.Error(), "insufficient content") {
		t.Errorf("expected 'insufficient content' in error, got: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// 6. Validate — missing required section
// ---------------------------------------------------------------------------

func TestGateValidateMissingRequiredSection(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	// Has a section, but not the required "Changes" section.
	err := gate.Validate("## Summary\nImplemented the OAuth2 login flow with full support for JWT token refresh and PKCE verification.")
	if err == nil {
		t.Fatal("expected error for missing required section")
	}
	if !strings.Contains(err.Error(), "missing required section") {
		t.Errorf("expected 'missing required section' in error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Changes") {
		t.Errorf("expected 'Changes' in error, got: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// 7. Validate — passes with valid evidence (file paths included)
// ---------------------------------------------------------------------------

func TestGateValidatePassesCodeComplete(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	evidence := "## Changes\n- auth/handler.go (added login endpoint)\n- auth/service.go (token generation logic)\n"
	err := gate.Validate(evidence)
	if err != nil {
		t.Fatalf("expected valid evidence to pass, got: %v", err)
	}
}

func TestGateValidatePassesTestComplete(t *testing.T) {
	gate := types.GetGate(types.StatusInTesting, types.StatusInDocs)
	evidence := "## Results\nAll 12 test cases pass. See auth/handler_test.go for details.\n"
	err := gate.Validate(evidence)
	if err != nil {
		t.Fatalf("expected valid test evidence to pass, got: %v", err)
	}
}

func TestGateValidatePassesDocsComplete(t *testing.T) {
	gate := types.GetGate(types.StatusInDocs, types.StatusInReview)
	evidence := "## Docs\nAdded docs/api/auth.md with full endpoint documentation.\n"
	err := gate.Validate(evidence)
	if err != nil {
		t.Fatalf("expected valid docs evidence to pass, got: %v", err)
	}
}

func TestGateValidateCaseInsensitiveSections(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	evidence := "## changes\n- auth/handler.go (added login endpoint)\n- auth/service.go (token generation)\n"
	err := gate.Validate(evidence)
	if err != nil {
		t.Fatalf("expected case-insensitive section matching to pass, got: %v", err)
	}
}

func TestGateValidateRejectsEvidenceWithoutFilePaths(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	evidence := "## Changes\nAdded the login endpoint and token generation logic to the auth module\n"
	err := gate.Validate(evidence)
	if err == nil {
		t.Fatal("expected error for Changes section without file paths")
	}
	if !strings.Contains(err.Error(), "file path") {
		t.Errorf("expected 'file path' in error, got: %s", err.Error())
	}
}

func TestGateValidateDocsRejectsWithoutFilePaths(t *testing.T) {
	gate := types.GetGate(types.StatusInDocs, types.StatusInReview)
	evidence := "## Docs\nThe documentation has been written and covers all endpoints thoroughly.\n"
	err := gate.Validate(evidence)
	if err == nil {
		t.Fatal("expected error for Docs section without file paths")
	}
	if !strings.Contains(err.Error(), "file path") {
		t.Errorf("expected 'file path' in error, got: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// 8. CheckFileTypes — test gate (matching and non-matching patterns)
// ---------------------------------------------------------------------------

func TestCheckFileTypesTestGateMatching(t *testing.T) {
	gate := types.GetGate(types.StatusInTesting, types.StatusInDocs)
	if gate == nil {
		t.Fatal("expected gate for in-testing -> in-docs")
	}

	matching := []string{
		"## Results\nAll tests pass. See auth/handler_test.go for verification.\n",
		"## Results\nFull coverage. Check src/auth.test.ts for unit tests.\n",
		"## Results\nIntegration tests at features/login.spec.ts pass.\n",
		"## Results\nRust tests pass: src/parser_test.rs confirmed correct.\n",
		"## Results\nJS tests pass: src/utils.test.js verified output.\n",
		"## Results\nPython tests pass: tests/test_auth.test.py verified.\n",
	}
	for _, evidence := range matching {
		ok, expected := gate.CheckFileTypes(evidence)
		if !ok {
			t.Errorf("expected match for evidence %q, wanted patterns %v", evidence, expected)
		}
	}
}

func TestCheckFileTypesTestGateNonMatching(t *testing.T) {
	gate := types.GetGate(types.StatusInTesting, types.StatusInDocs)
	if gate == nil {
		t.Fatal("expected gate for in-testing -> in-docs")
	}

	// File paths that do not match any test file pattern.
	evidence := "## Results\nVerified output by running src/handler.go manually.\n"
	ok, expected := gate.CheckFileTypes(evidence)
	if ok {
		t.Error("expected non-match for evidence referencing only .go (not _test.go)")
	}
	if len(expected) == 0 {
		t.Error("expected non-empty expected patterns on mismatch")
	}
}

func TestCheckFileTypesNoPatterns(t *testing.T) {
	// Code complete gate has no FilePatterns and no DocsFolder.
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	if gate == nil {
		t.Fatal("expected gate for in-progress -> in-testing")
	}

	ok, expected := gate.CheckFileTypes("## Changes\n- src/main.go updated\n")
	if !ok {
		t.Errorf("gate with no patterns should always match, got expected=%v", expected)
	}
	if expected != nil {
		t.Errorf("expected nil patterns slice, got %v", expected)
	}
}

func TestCheckFileTypesNoPaths(t *testing.T) {
	gate := types.GetGate(types.StatusInTesting, types.StatusInDocs)
	// Evidence with the right section but no file paths at all.
	evidence := "## Results\nAll tests pass with full coverage across the module.\n"
	ok, expected := gate.CheckFileTypes(evidence)
	if ok {
		t.Error("expected non-match when no file paths are present")
	}
	if len(expected) == 0 {
		t.Error("expected non-empty expected patterns")
	}
}

// ---------------------------------------------------------------------------
// 9. CheckFileTypes — docs gate (matching and non-matching patterns)
// ---------------------------------------------------------------------------

func TestCheckFileTypesDocsGateMatching(t *testing.T) {
	gate := types.GetGate(types.StatusInDocs, types.StatusInReview)
	if gate == nil {
		t.Fatal("expected gate for in-docs -> in-review")
	}

	evidence := "## Docs\nAdded endpoint documentation at docs/api/auth.md with examples.\n"
	ok, _ := gate.CheckFileTypes(evidence)
	if !ok {
		t.Error("expected match for docs/api/auth.md under docs/ folder")
	}
}

func TestCheckFileTypesDocsGateNonMatching(t *testing.T) {
	gate := types.GetGate(types.StatusInDocs, types.StatusInReview)
	if gate == nil {
		t.Fatal("expected gate for in-docs -> in-review")
	}

	// File not under docs/ folder.
	evidence := "## Docs\nUpdated README.md with the new auth setup instructions.\n"
	ok, expected := gate.CheckFileTypes(evidence)
	if ok {
		t.Error("expected non-match for file not under docs/ folder")
	}
	if len(expected) == 0 {
		t.Error("expected non-empty expected patterns on mismatch")
	}
}

func TestCheckFileTypesDocsGateNonMdFile(t *testing.T) {
	gate := types.GetGate(types.StatusInDocs, types.StatusInReview)
	if gate == nil {
		t.Fatal("expected gate for in-docs -> in-review")
	}

	// File under docs/ but not a .md file.
	evidence := "## Docs\nAdded docs/diagrams/arch.png for architecture diagram.\n"
	ok, expected := gate.CheckFileTypes(evidence)
	if ok {
		t.Error("expected non-match for non-.md file under docs/")
	}
	if len(expected) == 0 {
		t.Error("expected non-empty expected patterns on mismatch")
	}
}

// ---------------------------------------------------------------------------
// 10. ExtractFilePaths
// ---------------------------------------------------------------------------

func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "go files",
			text:     "Modified auth/handler.go and auth/service.go",
			expected: []string{"auth/handler.go", "auth/service.go"},
		},
		{
			name:     "markdown list",
			text:     "- libs/sdk-go/types/gates.go (added validation)\n- libs/sdk-go/types/gates_test.go (new tests)",
			expected: []string{"libs/sdk-go/types/gates.go", "libs/sdk-go/types/gates_test.go"},
		},
		{
			name:     "relative paths",
			text:     "Updated ./src/main.go and ../config.yaml",
			expected: []string{"./src/main.go", "../config.yaml"},
		},
		{
			name:     "deduplication",
			text:     "Changed auth/handler.go twice, see auth/handler.go for details",
			expected: []string{"auth/handler.go"},
		},
		{
			name:     "no paths",
			text:     "All tests pass with full coverage",
			expected: nil,
		},
		{
			name:     "various extensions",
			text:     "Updated src/app.ts, src/style.css, docs/README.md",
			expected: []string{"src/app.ts", "src/style.css", "docs/README.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := types.ExtractFilePaths(tt.text)
			if len(got) != len(tt.expected) {
				t.Fatalf("ExtractFilePaths() returned %d paths %v, want %d paths %v",
					len(got), got, len(tt.expected), tt.expected)
			}
			for i, path := range got {
				if path != tt.expected[i] {
					t.Errorf("path[%d] = %q, want %q", i, path, tt.expected[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 11. IsSkippableFor — docs gate skippable for bug/hotfix/testcase
// ---------------------------------------------------------------------------

func TestDocsGateSkippableForBugHotfixTestcase(t *testing.T) {
	// The in-docs -> in-review gate (DocsComplete) should be skippable.
	docsGate := types.GetGate(types.StatusInDocs, types.StatusInReview)
	if docsGate == nil {
		t.Fatal("expected gate for in-docs -> in-review")
	}

	skippable := []types.FeatureKind{types.KindBug, types.KindHotfix, types.KindTestcase}
	for _, kind := range skippable {
		if !docsGate.IsSkippableFor(kind) {
			t.Errorf("docs gate should be skippable for kind %q", kind)
		}
	}

	notSkippable := []types.FeatureKind{types.KindFeature, types.KindChore}
	for _, kind := range notSkippable {
		if docsGate.IsSkippableFor(kind) {
			t.Errorf("docs gate should NOT be skippable for kind %q", kind)
		}
	}
}

func TestTestCompleteSkipDocsGateSkippable(t *testing.T) {
	// The in-testing -> in-review gate (skip docs path) should also be skippable.
	gate := types.GetGate(types.StatusInTesting, types.StatusInReview)
	if gate == nil {
		t.Fatal("expected gate for in-testing -> in-review")
	}

	skippable := []types.FeatureKind{types.KindBug, types.KindHotfix, types.KindTestcase}
	for _, kind := range skippable {
		if !gate.IsSkippableFor(kind) {
			t.Errorf("test-complete skip-docs gate should be skippable for kind %q", kind)
		}
	}

	if gate.IsSkippableFor(types.KindFeature) {
		t.Error("test-complete skip-docs gate should NOT be skippable for kind 'feature'")
	}
}

func TestCodeCompleteGateNotSkippable(t *testing.T) {
	gate := types.GetGate(types.StatusInProgress, types.StatusInTesting)
	if gate == nil {
		t.Fatal("expected gate for in-progress -> in-testing")
	}

	allKinds := []types.FeatureKind{
		types.KindFeature, types.KindBug, types.KindHotfix,
		types.KindChore, types.KindTestcase,
	}
	for _, kind := range allKinds {
		if gate.IsSkippableFor(kind) {
			t.Errorf("code-complete gate should NOT be skippable for any kind, but was for %q", kind)
		}
	}
}

// ---------------------------------------------------------------------------
// Structural: every gate in the map has required fields populated
// ---------------------------------------------------------------------------

func TestAllGatesHaveRequiredFields(t *testing.T) {
	for from, toMap := range types.GateRequirements {
		for to, gate := range toMap {
			label := string(from) + " -> " + string(to)
			if gate.Name == "" {
				t.Errorf("gate %s has empty Name", label)
			}
			if gate.ID == "" {
				t.Errorf("gate %s has empty ID", label)
			}
			if gate.RequiredSection == "" {
				t.Errorf("gate %s has empty RequiredSection", label)
			}
			if gate.From != from {
				t.Errorf("gate %s has From=%s, want %s", label, gate.From, from)
			}
			if gate.To != to {
				t.Errorf("gate %s has To=%s, want %s", label, gate.To, to)
			}
		}
	}
}
