package types

// HypothesisStatus represents the validation state of a hypothesis.
type HypothesisStatus string

const (
	HypoUntested    HypothesisStatus = "untested"
	HypoTesting     HypothesisStatus = "testing"
	HypoValidated   HypothesisStatus = "validated"
	HypoInvalidated HypothesisStatus = "invalidated"
	HypoRefined     HypothesisStatus = "refined"
)

// ValidHypothesisStatuses lists all valid hypothesis statuses.
var ValidHypothesisStatuses = []string{"untested", "testing", "validated", "invalidated", "refined"}

// HypothesisData represents a user-problem hypothesis in the discovery spine.
type HypothesisData struct {
	ID          string           `json:"id"`
	ProjectID   string           `json:"project_id"`
	Title       string           `json:"title"`
	Problem     string           `json:"problem"`
	TargetUser  string           `json:"target_user"`
	Assumption  string           `json:"assumption"`
	Status      HypothesisStatus `json:"status"`
	CycleID     string           `json:"cycle_id,omitempty"`
	Experiments []string         `json:"experiments,omitempty"`
	RefinedFrom string           `json:"refined_from,omitempty"`
	Labels      []string         `json:"labels,omitempty"`
	Version     int64            `json:"version"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}
