package types

// DiscoveryReviewDecision captures a per-item decision in a discovery review.
type DiscoveryReviewDecision string

const (
	DecisionContinue DiscoveryReviewDecision = "continue"
	DecisionRefine   DiscoveryReviewDecision = "refine"
	DecisionPivot    DiscoveryReviewDecision = "pivot"
	DecisionStop     DiscoveryReviewDecision = "stop"
)

// ValidReviewDecisions lists all valid discovery review decisions.
var ValidReviewDecisions = []string{"continue", "refine", "pivot", "stop"}

// DiscoveryReviewItem captures a decision about one hypothesis or experiment.
type DiscoveryReviewItem struct {
	ItemID    string                  `json:"item_id"`
	ItemType  string                  `json:"item_type"`
	Decision  DiscoveryReviewDecision `json:"decision"`
	Rationale string                  `json:"rationale"`
}

// DiscoveryReviewData represents a weekly discovery review session.
type DiscoveryReviewData struct {
	ID              string                `json:"id"`
	ProjectID       string                `json:"project_id"`
	CycleID         string                `json:"cycle_id"`
	Title           string                `json:"title"`
	Surprises       string                `json:"surprises,omitempty"`
	WrongAbout      string                `json:"wrong_about,omitempty"`
	Items           []DiscoveryReviewItem `json:"items,omitempty"`
	TransitionReady bool                  `json:"transition_ready,omitempty"`
	Version         int64                 `json:"version"`
	CreatedAt       string                `json:"created_at"`
	UpdatedAt       string                `json:"updated_at"`
}
