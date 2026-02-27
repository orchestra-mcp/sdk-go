package types

// ValidTransitions defines the allowed state transitions in the feature-driven
// cyclical workflow. Each key maps to the set of statuses it can transition to.
var ValidTransitions = map[FeatureStatus][]FeatureStatus{
	StatusBacklog:         {StatusTodo},
	StatusTodo:            {StatusInProgress},
	StatusInProgress:      {StatusReadyForTesting},
	StatusReadyForTesting: {StatusInTesting},
	StatusInTesting:       {StatusReadyForDocs, StatusInProgress},
	StatusReadyForDocs:    {StatusInDocs},
	StatusInDocs:          {StatusDocumented},
	StatusDocumented:      {StatusInReview},
	StatusInReview:        {StatusDone, StatusNeedsEdits},
	StatusNeedsEdits:      {StatusInProgress},
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
