package types

// ProjectMode represents the operating mode of a project.
type ProjectMode string

const (
	ModeDiscovery ProjectMode = "discovery"
	ModeOutcome   ProjectMode = "outcome"
	ModeScale     ProjectMode = "scale"
)

// ValidModes lists all valid project modes.
var ValidModes = []string{"discovery", "outcome", "scale"}

// DiscoveryCycleStatus represents the lifecycle state of a discovery cycle.
type DiscoveryCycleStatus string

const (
	CycleActive    DiscoveryCycleStatus = "active"
	CycleCompleted DiscoveryCycleStatus = "completed"
	CycleCancelled DiscoveryCycleStatus = "cancelled"
)

// ValidCycleStatuses lists all valid discovery cycle statuses.
var ValidCycleStatuses = []string{"active", "completed", "cancelled"}

// DiscoveryCycleData represents a time-boxed discovery cycle (1-2 weeks).
type DiscoveryCycleData struct {
	ID          string               `json:"id"`
	ProjectID   string               `json:"project_id"`
	Title       string               `json:"title"`
	Goal        string               `json:"goal"`
	StartDate   string               `json:"start_date"`
	EndDate     string               `json:"end_date"`
	Status      DiscoveryCycleStatus `json:"status"`
	Hypotheses  []string             `json:"hypotheses,omitempty"`
	Experiments []string             `json:"experiments,omitempty"`
	Learnings   string               `json:"learnings,omitempty"`
	Decision    string               `json:"decision,omitempty"`
	Version     int64                `json:"version"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}
