package types

// RequestStatus represents the lifecycle state of a user request.
type RequestStatus string

const (
	RequestPending   RequestStatus = "pending"
	RequestPickedUp  RequestStatus = "picked-up"
	RequestConverted RequestStatus = "converted"
	RequestDismissed RequestStatus = "dismissed"
)

// RequestData represents a user request queued while the agent is busy.
type RequestData struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"project_id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Kind        string        `json:"kind"`                    // feature, hotfix, bug
	Status      RequestStatus `json:"status"`
	Priority    string        `json:"priority"`
	ConvertedTo string        `json:"converted_to,omitempty"` // feature ID if converted
	Version     int64         `json:"version"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
}
