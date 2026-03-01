package types

// PlanStatus represents the lifecycle state of a plan.
type PlanStatus string

const (
	PlanDraft      PlanStatus = "draft"
	PlanApproved   PlanStatus = "approved"
	PlanInProgress PlanStatus = "in-progress"
	PlanCompleted  PlanStatus = "completed"
)

// PlanData represents a plan that breaks down into multiple features.
type PlanData struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      PlanStatus `json:"status"`
	Features    []string   `json:"features,omitempty"` // linked feature IDs after breakdown
	Version     int64      `json:"version"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}
