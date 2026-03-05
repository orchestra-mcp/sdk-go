package types

// PersonRole defines the role of a person within a project.
type PersonRole string

const (
	RoleDeveloper PersonRole = "developer"
	RoleQA        PersonRole = "qa"
	RoleReviewer  PersonRole = "reviewer"
	RoleLead      PersonRole = "lead"
)

// ValidRoles is the set of valid PersonRole values for validation.
var ValidRoles = []string{"developer", "qa", "reviewer", "lead"}

// PersonStatus represents whether a person is active.
type PersonStatus string

const (
	PersonActive   PersonStatus = "active"
	PersonInactive PersonStatus = "inactive"
)

// ValidPersonStatuses is the set of valid PersonStatus values for validation.
var ValidPersonStatuses = []string{"active", "inactive"}

// PersonData represents a person within a project.
type PersonData struct {
	ID        string       `json:"id"`
	ProjectID string       `json:"project_id"`
	Name      string       `json:"name"`
	Email     string       `json:"email,omitempty"`
	Role      PersonRole   `json:"role"`
	Status    PersonStatus `json:"status"`
	Labels    []string     `json:"labels,omitempty"`
	Version   int64        `json:"version"`
	CreatedAt string       `json:"created_at"`
	UpdatedAt string       `json:"updated_at"`

	// Profile fields
	Bio          string            `json:"bio,omitempty"`
	GithubEmail  string            `json:"github_email,omitempty"`
	Integrations map[string]string `json:"integrations,omitempty"`
}
