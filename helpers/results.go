package helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/types"
	"google.golang.org/protobuf/types/known/structpb"
)

// TextResult creates a successful ToolResponse containing a text result.
func TextResult(text string) *pluginv1.ToolResponse {
	s, _ := structpb.NewStruct(map[string]any{
		"text": text,
	})
	return &pluginv1.ToolResponse{
		Success: true,
		Result:  s,
	}
}

// JSONResult creates a successful ToolResponse containing the given data
// marshaled as a structpb.Struct. The data must be JSON-serializable.
func JSONResult(data any) (*pluginv1.ToolResponse, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		// If the data is not a map, wrap it in one.
		m = map[string]any{"data": data}
		raw2, _ := json.Marshal(m)
		if err2 := json.Unmarshal(raw2, &m); err2 != nil {
			return nil, fmt.Errorf("convert to struct: %w", err2)
		}
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil, fmt.Errorf("new struct: %w", err)
	}
	return &pluginv1.ToolResponse{
		Success: true,
		Result:  s,
	}, nil
}

// ErrorResult creates a failed ToolResponse with the given error code and message.
func ErrorResult(code string, message string) *pluginv1.ToolResponse {
	return &pluginv1.ToolResponse{
		Success:      false,
		ErrorCode:    code,
		ErrorMessage: message,
	}
}

// --- Markdown formatters ---

// FormatFeatureMD formats a single feature as a Markdown block.
func FormatFeatureMD(f *types.FeatureData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s\n", f.ID, f.Title)
	fmt.Fprintf(&b, "- **Status:** %s\n", f.Status)
	fmt.Fprintf(&b, "- **Priority:** %s\n", f.Priority)
	if f.Kind != "" {
		fmt.Fprintf(&b, "- **Kind:** %s\n", f.Kind)
	}
	if f.Assignee != "" {
		fmt.Fprintf(&b, "- **Assignee:** %s\n", f.Assignee)
	}
	if f.Estimate != "" {
		fmt.Fprintf(&b, "- **Estimate:** %s\n", f.Estimate)
	}
	if len(f.Labels) > 0 {
		fmt.Fprintf(&b, "- **Labels:** %s\n", strings.Join(f.Labels, ", "))
	}
	if f.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", f.Description)
	}
	return b.String()
}

// FormatFeatureListMD formats a list of features as a Markdown table.
func FormatFeatureListMD(features []*types.FeatureData, header string) string {
	if len(features) == 0 {
		return fmt.Sprintf("## %s\n\nNo features found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(features))
	fmt.Fprintf(&b, "| ID | Title | Status | Priority | Kind | Assignee |\n")
	fmt.Fprintf(&b, "|----|-------|--------|----------|------|----------|\n")
	for _, f := range features {
		assignee := f.Assignee
		if assignee == "" {
			assignee = "—"
		}
		kind := string(f.Kind)
		if kind == "" {
			kind = "feature"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", f.ID, f.Title, f.Status, f.Priority, kind, assignee)
	}
	return b.String()
}

// FormatPlanMD formats a single plan as a Markdown block.
func FormatPlanMD(p *types.PlanData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s\n", p.ID, p.Title)
	fmt.Fprintf(&b, "- **Status:** %s\n", p.Status)
	if len(p.Features) > 0 {
		fmt.Fprintf(&b, "- **Features:** %s\n", strings.Join(p.Features, ", "))
	}
	if p.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", p.Description)
	}
	return b.String()
}

// FormatPlanListMD formats a list of plans as a Markdown table.
func FormatPlanListMD(plans []*types.PlanData, header string) string {
	if len(plans) == 0 {
		return fmt.Sprintf("## %s\n\nNo plans found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(plans))
	fmt.Fprintf(&b, "| ID | Title | Status | Features |\n")
	fmt.Fprintf(&b, "|----|-------|--------|----------|\n")
	for _, p := range plans {
		featureCount := fmt.Sprintf("%d", len(p.Features))
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", p.ID, p.Title, p.Status, featureCount)
	}
	return b.String()
}

// FormatRequestMD formats a single request as a Markdown block.
func FormatRequestMD(r *types.RequestData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s\n", r.ID, r.Title)
	fmt.Fprintf(&b, "- **Status:** %s\n", r.Status)
	fmt.Fprintf(&b, "- **Kind:** %s\n", r.Kind)
	fmt.Fprintf(&b, "- **Priority:** %s\n", r.Priority)
	if r.ConvertedTo != "" {
		fmt.Fprintf(&b, "- **Converted to:** %s\n", r.ConvertedTo)
	}
	if r.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", r.Description)
	}
	return b.String()
}

// FormatRequestListMD formats a list of requests as a Markdown table.
func FormatRequestListMD(requests []*types.RequestData, header string) string {
	if len(requests) == 0 {
		return fmt.Sprintf("## %s\n\nNo requests found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(requests))
	fmt.Fprintf(&b, "| ID | Title | Kind | Status | Priority |\n")
	fmt.Fprintf(&b, "|----|-------|------|--------|----------|\n")
	for _, r := range requests {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", r.ID, r.Title, r.Kind, r.Status, r.Priority)
	}
	return b.String()
}

// FormatProjectMD formats a project as Markdown.
func FormatProjectMD(p *types.ProjectData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Project: %s\n", p.Name)
	fmt.Fprintf(&b, "- **Slug:** %s\n", p.Slug)
	if p.Description != "" {
		fmt.Fprintf(&b, "- **Description:** %s\n", p.Description)
	}
	fmt.Fprintf(&b, "- **Created:** %s\n", p.CreatedAt)
	return b.String()
}

// FormatProjectListMD formats a list of projects as Markdown.
func FormatProjectListMD(projects []*types.ProjectData) string {
	if len(projects) == 0 {
		return "## Projects\n\nNo projects found.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Projects (%d)\n\n", len(projects))
	for _, p := range projects {
		desc := ""
		if p.Description != "" {
			desc = " — " + p.Description
		}
		fmt.Fprintf(&b, "- **%s** (`%s`)%s\n", p.Name, p.Slug, desc)
	}
	return b.String()
}

// FormatPersonMD formats a single person as a Markdown block.
func FormatPersonMD(p *types.PersonData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s\n", p.ID, p.Name)
	fmt.Fprintf(&b, "- **Role:** %s\n", p.Role)
	fmt.Fprintf(&b, "- **Status:** %s\n", p.Status)
	if p.Email != "" {
		fmt.Fprintf(&b, "- **Email:** %s\n", p.Email)
	}
	if p.Bio != "" {
		fmt.Fprintf(&b, "- **Bio:** %s\n", p.Bio)
	}
	if p.GithubEmail != "" {
		fmt.Fprintf(&b, "- **GitHub Email:** %s\n", p.GithubEmail)
	}
	if len(p.Integrations) > 0 {
		for k, v := range p.Integrations {
			fmt.Fprintf(&b, "- **%s:** %s\n", k, v)
		}
	}
	if len(p.Labels) > 0 {
		fmt.Fprintf(&b, "- **Labels:** %s\n", strings.Join(p.Labels, ", "))
	}
	return b.String()
}

// FormatPersonListMD formats a list of persons as a Markdown table.
func FormatPersonListMD(persons []*types.PersonData, header string) string {
	if len(persons) == 0 {
		return fmt.Sprintf("## %s\n\nNo persons found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(persons))
	fmt.Fprintf(&b, "| ID | Name | Role | Status | Email |\n")
	fmt.Fprintf(&b, "|----|------|------|--------|-------|\n")
	for _, p := range persons {
		email := p.Email
		if email == "" {
			email = "—"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", p.ID, p.Name, p.Role, p.Status, email)
	}
	return b.String()
}

// FormatAssignmentRuleMD formats a single assignment rule as a Markdown block.
func FormatAssignmentRuleMD(r *types.AssignmentRuleData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n", r.ID)
	fmt.Fprintf(&b, "- **Kind:** %s\n", r.Kind)
	fmt.Fprintf(&b, "- **Assigned to:** %s\n", r.PersonID)
	return b.String()
}

// FormatAssignmentRuleListMD formats a list of assignment rules as a Markdown table.
func FormatAssignmentRuleListMD(rules []*types.AssignmentRuleData, header string) string {
	if len(rules) == 0 {
		return fmt.Sprintf("## %s\n\nNo assignment rules found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(rules))
	fmt.Fprintf(&b, "| ID | Kind | Person ID |\n")
	fmt.Fprintf(&b, "|----|------|-----------|\n")
	for _, r := range rules {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", r.ID, r.Kind, r.PersonID)
	}
	return b.String()
}

// FormatStatusCountsMD formats status counts as a Markdown table.
func FormatStatusCountsMD(counts map[string]int, total int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "| Status | Count |\n")
	fmt.Fprintf(&b, "|--------|-------|\n")
	for status, count := range counts {
		fmt.Fprintf(&b, "| %s | %d |\n", status, count)
	}
	fmt.Fprintf(&b, "| **Total** | **%d** |\n", total)
	return b.String()
}
