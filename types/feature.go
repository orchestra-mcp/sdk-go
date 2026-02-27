package types

// FeatureStatus represents the current state of a feature in the workflow.
type FeatureStatus string

const (
	StatusBacklog         FeatureStatus = "backlog"
	StatusTodo            FeatureStatus = "todo"
	StatusInProgress      FeatureStatus = "in-progress"
	StatusReadyForTesting FeatureStatus = "ready-for-testing"
	StatusInTesting       FeatureStatus = "in-testing"
	StatusReadyForDocs    FeatureStatus = "ready-for-docs"
	StatusInDocs          FeatureStatus = "in-docs"
	StatusDocumented      FeatureStatus = "documented"
	StatusInReview        FeatureStatus = "in-review"
	StatusNeedsEdits      FeatureStatus = "needs-edits"
	StatusDone            FeatureStatus = "done"
)

// FeatureData represents a feature within a project.
type FeatureData struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"project_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      FeatureStatus `json:"status"`
	Priority    string        `json:"priority"` // P0-P3
	Assignee    string        `json:"assignee,omitempty"`
	Labels      []string      `json:"labels,omitempty"`
	DependsOn   []string      `json:"depends_on,omitempty"`
	Blocks      []string      `json:"blocks,omitempty"`
	Estimate    string        `json:"estimate,omitempty"` // S/M/L/XL
	Version     int64         `json:"version"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
}

// ReviewEntry records a review decision on a feature.
type ReviewEntry struct {
	Reviewer  string `json:"reviewer"`
	Status    string `json:"status"` // approved, needs-edits, rejected
	Comment   string `json:"comment"`
	CreatedAt string `json:"created_at"`
}
