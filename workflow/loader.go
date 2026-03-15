package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads a WorkflowDefinition from a YAML file at the given path.
func LoadFromFile(path string) (*WorkflowDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow file %s: %w", path, err)
	}
	var def WorkflowDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse workflow file %s: %w", path, err)
	}
	return &def, nil
}

// LoadFromDir loads the first *.yaml file found in dir as a WorkflowDefinition.
// Returns nil, nil if the directory does not exist or contains no YAML files —
// the caller should fall back to DefaultEngine() in that case.
func LoadFromDir(dir string) (*WorkflowDefinition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workflow dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".yaml" || filepath.Ext(name) == ".yml" {
			return LoadFromFile(filepath.Join(dir, name))
		}
	}
	return nil, nil
}

// DefaultDefinition returns the built-in 7-state Orchestra feature workflow.
// This exactly mirrors the hardcoded ValidTransitions and GateRequirements maps
// in libs/sdk-go/types/workflow.go and types/gates.go.
func DefaultDefinition() WorkflowDefinition {
	return WorkflowDefinition{
		Name:         "orchestra-default",
		Description:  "Default 7-state Orchestra feature delivery workflow",
		InitialState: "todo",
		States: map[StateID]StateDef{
			"todo": {
				Label:      "To Do",
				Terminal:   false,
				ActiveWork: false,
			},
			"in-progress": {
				Label:      "In Progress",
				Terminal:   false,
				ActiveWork: true,
			},
			"in-testing": {
				Label:      "In Testing",
				Terminal:   false,
				ActiveWork: true,
			},
			"in-docs": {
				Label:      "In Docs",
				Terminal:   false,
				ActiveWork: true,
			},
			"in-review": {
				Label:      "In Review",
				Terminal:   false,
				ActiveWork: true,
			},
			"needs-edits": {
				Label:      "Needs Edits",
				Terminal:   false,
				ActiveWork: false,
			},
			"done": {
				Label:    "Done",
				Terminal: true,
			},
		},
		// Transitions are ordered: the first target for each "from" state is the
		// default next step. Multiple targets give the engine context (e.g. in-testing
		// can go to in-docs OR in-review for bugs/hotfixes).
		Transitions: []TransitionDef{
			{From: "todo", To: "in-progress"},
			{From: "in-progress", To: "in-testing", Gate: "code_complete"},
			{From: "in-testing", To: "in-docs", Gate: "test_complete"},
			{From: "in-testing", To: "in-review", Gate: "test_complete_skip_docs"},
			{From: "in-docs", To: "in-review", Gate: "docs_complete"},
			{From: "in-review", To: "done"},
			{From: "in-review", To: "needs-edits"},
			{From: "needs-edits", To: "in-progress"},
		},
		Gates: map[string]GateDef{
			"code_complete": {
				Label:           "Code Complete",
				RequiredSection: "Changes",
				FilePatterns:    []string{},
				SkippableFor:    []string{},
			},
			"test_complete": {
				Label:           "Test Complete",
				RequiredSection: "Results",
				FilePatterns: []string{
					"_test.go",
					".test.ts",
					".test.tsx",
					".spec.ts",
					".spec.tsx",
					"_test.rs",
					".test.js",
					".spec.js",
					".test.py",
					"_test.py",
				},
				SkippableFor: []string{},
			},
			// test_complete_skip_docs is used when transitioning in-testing → in-review
			// directly (for bugs, hotfixes, testcases that skip the docs phase).
			"test_complete_skip_docs": {
				Label:           "Test Complete (skip docs)",
				RequiredSection: "Results",
				FilePatterns: []string{
					"_test.go",
					".test.ts",
					".test.tsx",
					".spec.ts",
					".spec.tsx",
					"_test.rs",
					".test.js",
					".spec.js",
					".test.py",
					"_test.py",
				},
				SkippableFor: []string{"bug", "hotfix", "testcase"},
			},
			"docs_complete": {
				Label:           "Docs Complete",
				RequiredSection: "Docs",
				DocsFolder:      "docs",
				SkippableFor:    []string{"bug", "hotfix", "testcase"},
			},
		},
	}
}

// DefaultEngine returns an Engine built from DefaultDefinition().
// Use this as the fallback when no pack overrides a custom workflow.
func DefaultEngine() *Engine {
	def := DefaultDefinition()
	return NewEngine(def)
}
