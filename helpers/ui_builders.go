package helpers

import "github.com/orchestra-mcp/sdk-go/types"

// --- Feature ---

// FeatureDetailUI creates UIMetadata for a single feature detail card.
func FeatureDetailUI(f *types.FeatureData, actions ...UIAction) *UIMetadata {
	return &UIMetadata{
		View:       ViewDetailCard,
		EntityType: "feature",
		Data:       f,
		Actions:    actions,
	}
}

// FeatureListUI creates UIMetadata for a feature list table.
func FeatureListUI(features []*types.FeatureData, pg PaginationParams, total int, actions ...UIAction) *UIMetadata {
	return &UIMetadata{
		View:       ViewTable,
		EntityType: "feature",
		Data:       features,
		Columns: []UIColumn{
			{Key: "id", Label: "ID", Type: "text"},
			{Key: "title", Label: "Title", Type: "text"},
			{Key: "status", Label: "Status", Type: "status"},
			{Key: "priority", Label: "Priority", Type: "priority"},
			{Key: "kind", Label: "Kind", Type: "badge"},
			{Key: "assignee", Label: "Assignee", Type: "text"},
		},
		Pagination: &UIPagination{Total: total, Limit: pg.Limit, Offset: pg.Offset},
		Actions:    actions,
	}
}

// --- Plan ---

// PlanDetailUI creates UIMetadata for a single plan detail card.
func PlanDetailUI(p *types.PlanData, actions ...UIAction) *UIMetadata {
	return &UIMetadata{
		View:       ViewDetailCard,
		EntityType: "plan",
		Data:       p,
		Actions:    actions,
	}
}

// PlanListUI creates UIMetadata for a plan list table.
func PlanListUI(plans []*types.PlanData, pg PaginationParams, total int) *UIMetadata {
	return &UIMetadata{
		View:       ViewTable,
		EntityType: "plan",
		Data:       plans,
		Columns: []UIColumn{
			{Key: "id", Label: "ID", Type: "text"},
			{Key: "title", Label: "Title", Type: "text"},
			{Key: "status", Label: "Status", Type: "status"},
			{Key: "features", Label: "Features", Type: "text"},
		},
		Pagination: &UIPagination{Total: total, Limit: pg.Limit, Offset: pg.Offset},
	}
}

// --- Person ---

// PersonDetailUI creates UIMetadata for a single person detail card.
func PersonDetailUI(p *types.PersonData, actions ...UIAction) *UIMetadata {
	return &UIMetadata{
		View:       ViewDetailCard,
		EntityType: "person",
		Data:       p,
		Actions:    actions,
	}
}

// PersonListUI creates UIMetadata for a person list table.
func PersonListUI(persons []*types.PersonData, pg PaginationParams, total int) *UIMetadata {
	return &UIMetadata{
		View:       ViewTable,
		EntityType: "person",
		Data:       persons,
		Columns: []UIColumn{
			{Key: "id", Label: "ID", Type: "text"},
			{Key: "name", Label: "Name", Type: "text"},
			{Key: "role", Label: "Role", Type: "badge"},
			{Key: "status", Label: "Status", Type: "status"},
			{Key: "email", Label: "Email", Type: "text"},
		},
		Pagination: &UIPagination{Total: total, Limit: pg.Limit, Offset: pg.Offset},
	}
}

// --- Request ---

// RequestDetailUI creates UIMetadata for a single request detail card.
func RequestDetailUI(r *types.RequestData, actions ...UIAction) *UIMetadata {
	return &UIMetadata{
		View:       ViewDetailCard,
		EntityType: "request",
		Data:       r,
		Actions:    actions,
	}
}

// RequestListUI creates UIMetadata for a request list table.
func RequestListUI(requests []*types.RequestData, pg PaginationParams, total int) *UIMetadata {
	return &UIMetadata{
		View:       ViewTable,
		EntityType: "request",
		Data:       requests,
		Columns: []UIColumn{
			{Key: "id", Label: "ID", Type: "text"},
			{Key: "title", Label: "Title", Type: "text"},
			{Key: "kind", Label: "Kind", Type: "badge"},
			{Key: "status", Label: "Status", Type: "status"},
			{Key: "priority", Label: "Priority", Type: "priority"},
		},
		Pagination: &UIPagination{Total: total, Limit: pg.Limit, Offset: pg.Offset},
	}
}

// --- Delegation ---

// DelegationDetailUI creates UIMetadata for a single delegation.
func DelegationDetailUI(d *types.DelegationData, actions ...UIAction) *UIMetadata {
	return &UIMetadata{
		View:       ViewDetailCard,
		EntityType: "delegation",
		Data:       d,
		Actions:    actions,
	}
}

// DelegationListUI creates UIMetadata for a delegation list table.
func DelegationListUI(delegations []*types.DelegationData, pg PaginationParams, total int) *UIMetadata {
	return &UIMetadata{
		View:       ViewTable,
		EntityType: "delegation",
		Data:       delegations,
		Columns: []UIColumn{
			{Key: "id", Label: "ID", Type: "text"},
			{Key: "feature_id", Label: "Feature", Type: "text"},
			{Key: "from_person", Label: "From", Type: "text"},
			{Key: "to_person", Label: "To", Type: "text"},
			{Key: "status", Label: "Status", Type: "status"},
		},
		Pagination: &UIPagination{Total: total, Limit: pg.Limit, Offset: pg.Offset},
	}
}

// --- Assignment Rule ---

// AssignmentRuleListUI creates UIMetadata for an assignment rule list table.
func AssignmentRuleListUI(rules []*types.AssignmentRuleData, pg PaginationParams, total int) *UIMetadata {
	return &UIMetadata{
		View:       ViewTable,
		EntityType: "assignment_rule",
		Data:       rules,
		Columns: []UIColumn{
			{Key: "id", Label: "ID", Type: "text"},
			{Key: "kind", Label: "Kind", Type: "badge"},
			{Key: "person_id", Label: "Person", Type: "text"},
		},
		Pagination: &UIPagination{Total: total, Limit: pg.Limit, Offset: pg.Offset},
	}
}

// --- Dashboard/Progress ---

// ProgressUI creates UIMetadata for a project progress dashboard.
func ProgressUI(statusCounts map[string]int, total, done int, pctDone float64) *UIMetadata {
	return &UIMetadata{
		View:       ViewDashboard,
		EntityType: "progress",
		Data: map[string]any{
			"total":         total,
			"done":          done,
			"percent_done":  pctDone,
			"status_counts": statusCounts,
		},
	}
}

// WorkflowKanbanUI creates UIMetadata for a kanban board view by status.
func WorkflowKanbanUI(features []*types.FeatureData, statusCounts map[string]int, total int) *UIMetadata {
	return &UIMetadata{
		View:       ViewKanban,
		EntityType: "feature",
		Data:       features,
		Groups: []UIGroup{
			{Key: "todo", Label: "Todo", Color: "#6B7280"},
			{Key: "in-progress", Label: "In Progress", Color: "#3B82F6"},
			{Key: "in-testing", Label: "In Testing", Color: "#F59E0B"},
			{Key: "in-docs", Label: "In Docs", Color: "#8B5CF6"},
			{Key: "in-review", Label: "In Review", Color: "#10B981"},
			{Key: "done", Label: "Done", Color: "#22C55E"},
			{Key: "needs-edits", Label: "Needs Edits", Color: "#EF4444"},
		},
	}
}

// --- Project ---

// ProjectDetailUI creates UIMetadata for a project detail card.
func ProjectDetailUI(data any, actions ...UIAction) *UIMetadata {
	return &UIMetadata{
		View:       ViewDetailCard,
		EntityType: "project",
		Data:       data,
		Actions:    actions,
	}
}

// --- Dependency Graph ---

// DependencyTreeUI creates UIMetadata for a dependency graph tree view.
func DependencyTreeUI(graph any) *UIMetadata {
	return &UIMetadata{
		View:       ViewTree,
		EntityType: "dependency",
		Data:       graph,
	}
}
