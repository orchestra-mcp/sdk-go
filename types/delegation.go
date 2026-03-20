package types

// DelegationStatus represents the lifecycle state of a delegation.
type DelegationStatus string

const (
	DelegationPending   DelegationStatus = "pending"
	DelegationAnswered  DelegationStatus = "answered"
	DelegationDismissed DelegationStatus = "dismissed"
)

// DelegationData represents a delegation request from one person to another.
type DelegationData struct {
	ID          string           `json:"id"`
	ProjectID   string           `json:"project_id"`
	FeatureID   string           `json:"feature_id"`
	FromPerson  string           `json:"from_person"`
	ToPerson    string           `json:"to_person"`
	Question    string           `json:"question"`
	Context     string           `json:"context,omitempty"`
	Response    string           `json:"response,omitempty"`
	Status      DelegationStatus `json:"status"`
	Version     int64            `json:"version"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
	RespondedAt string           `json:"responded_at,omitempty"`
}
