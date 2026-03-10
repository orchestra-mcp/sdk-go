package types

// ValidTransitions defines the allowed state transitions in the simplified
// feature workflow. Each status maps to exactly one activity:
//
//	todo         → not started
//	in-progress  → writing source code ONLY
//	in-testing   → writing test code and running tests ONLY
//	in-docs      → writing .md docs in /docs folder ONLY
//	in-review    → waiting for human approval ONLY
//	needs-edits  → rejected, loops back to in-progress
//	done         → complete (terminal)
var ValidTransitions = map[FeatureStatus][]FeatureStatus{
	StatusTodo:       {StatusInProgress},
	StatusInProgress: {StatusInTesting},
	StatusInTesting:  {StatusInDocs, StatusInReview}, // in-review for bug/hotfix (skip docs)
	StatusInDocs:     {StatusInReview},
	StatusInReview:   {StatusDone, StatusNeedsEdits},
	StatusNeedsEdits: {StatusInProgress},
}

// CanTransition reports whether a transition from one status to another is valid.
func CanTransition(from, to FeatureStatus) bool {
	targets, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// NextStatuses returns the list of valid next statuses from the given status.
// Returns nil if the status has no outgoing transitions (e.g., StatusDone).
func NextStatuses(from FeatureStatus) []FeatureStatus {
	return ValidTransitions[from]
}

// MigrateStatus maps legacy statuses to their new equivalents.
// Returns the status unchanged if it is already a valid new status.
func MigrateStatus(s FeatureStatus) FeatureStatus {
	if mapped, ok := LegacyStatusMap[s]; ok {
		return mapped
	}
	return s
}
