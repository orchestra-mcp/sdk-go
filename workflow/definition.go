// Package workflow provides a pluggable workflow engine for Orchestra feature
// lifecycles. The engine is defined by a WorkflowDefinition (loaded from YAML
// or code) and provides transition validation, gate lookup, and state metadata.
package workflow

// StateID is a workflow state identifier (e.g. "todo", "in-progress").
type StateID string

// WorkflowDefinition describes a complete feature lifecycle as a directed graph
// of states and gated transitions. It can be loaded from YAML or constructed in
// code via DefaultDefinition().
type WorkflowDefinition struct {
	Name         string               `yaml:"name"`
	Description  string               `yaml:"description,omitempty"`
	InitialState StateID              `yaml:"initial_state"`
	States       map[StateID]StateDef `yaml:"states"`
	Transitions  []TransitionDef      `yaml:"transitions"`
	Gates        map[string]GateDef   `yaml:"gates,omitempty"`
}

// StateDef describes a single workflow state.
type StateDef struct {
	// Label is the human-readable display name.
	Label string `yaml:"label"`
	// Terminal marks the state as a final state (no outgoing transitions, e.g. "done").
	Terminal bool `yaml:"terminal,omitempty"`
	// ActiveWork indicates the feature is being actively worked on (e.g. in-progress, in-testing).
	ActiveWork bool `yaml:"active_work,omitempty"`
}

// TransitionDef describes a valid state transition and its optional gate.
type TransitionDef struct {
	// From is the source state ID.
	From string `yaml:"from"`
	// To is the target state ID.
	To string `yaml:"to"`
	// Gate references a GateDef key in WorkflowDefinition.Gates. If empty,
	// the transition is free (no evidence required).
	Gate string `yaml:"gate,omitempty"`
}

// GateDef describes the evidence requirements for a gated workflow transition.
type GateDef struct {
	// Label is the human-readable gate name (e.g. "Code Complete").
	Label string `yaml:"label"`
	// RequiredSection is the markdown section header (without "## ") that must
	// be present in the evidence string (e.g. "Changes", "Results", "Docs").
	RequiredSection string `yaml:"required_section"`
	// FilePatterns lists glob/suffix patterns that at least one referenced file
	// must match. An empty list means any file path is accepted.
	FilePatterns []string `yaml:"file_patterns,omitempty"`
	// DocsFolder, if non-empty, requires that referenced file paths start with
	// this folder prefix and have a .md extension (e.g. "docs").
	DocsFolder string `yaml:"docs_folder,omitempty"`
	// SkippableFor lists feature kinds for which this gate is auto-passed
	// (e.g. ["bug", "hotfix"] skips the docs gate for those kinds).
	SkippableFor []string `yaml:"skippable_for,omitempty"`
}
