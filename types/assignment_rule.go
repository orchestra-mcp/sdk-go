package types

// AssignmentRuleData represents an auto-assignment rule. When a feature of the
// specified kind is created, it is automatically assigned to the given person.
type AssignmentRuleData struct {
	ID        string `json:"id"`         // RULE-XXX
	ProjectID string `json:"project_id"`
	Kind      string `json:"kind"`      // feature kind to match (feature/bug/hotfix/chore/testcase)
	PersonID  string `json:"person_id"` // PERS-XXX to assign to
	Version   int64  `json:"version"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
