package helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/orchestra-mcp/sdk-go/globaldb"
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

// FormatFeatureListMDWithLocks formats a feature list annotating any in-progress
// features that are locked by a different session with "[locked by other session]".
// This prevents a new session from mistakenly treating another session's active
// feature as its own work.
func FormatFeatureListMDWithLocks(features []*types.FeatureData, header, projectID, sessionID string) string {
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
		status := string(f.Status)
		// Annotate in-progress/in-testing/in-docs/in-review features locked by another session.
		if sessionID != "" && isActiveFeatureStatus(f.Status) {
			if lock, _ := globaldb.GetLockInfo(projectID, f.ID); lock != nil && lock.SessionID != sessionID {
				status = status + " [locked by other session]"
			}
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", f.ID, f.Title, status, f.Priority, kind, assignee)
	}
	return b.String()
}

// isActiveFeatureStatus returns true for statuses that can hold a session lock.
func isActiveFeatureStatus(s types.FeatureStatus) bool {
	switch s {
	case types.StatusInProgress, types.StatusInTesting, types.StatusInDocs, types.StatusInReview:
		return true
	}
	return false
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

// FormatHypothesisMD formats a single hypothesis as a Markdown block.
func FormatHypothesisMD(h *types.HypothesisData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s\n", h.ID, h.Title)
	fmt.Fprintf(&b, "- **Status:** %s\n", h.Status)
	fmt.Fprintf(&b, "- **Problem:** %s\n", h.Problem)
	fmt.Fprintf(&b, "- **Target User:** %s\n", h.TargetUser)
	fmt.Fprintf(&b, "- **Assumption:** %s\n", h.Assumption)
	if h.CycleID != "" {
		fmt.Fprintf(&b, "- **Cycle:** %s\n", h.CycleID)
	}
	if h.RefinedFrom != "" {
		fmt.Fprintf(&b, "- **Refined from:** %s\n", h.RefinedFrom)
	}
	if len(h.Experiments) > 0 {
		fmt.Fprintf(&b, "- **Experiments:** %s\n", strings.Join(h.Experiments, ", "))
	}
	if len(h.Labels) > 0 {
		fmt.Fprintf(&b, "- **Labels:** %s\n", strings.Join(h.Labels, ", "))
	}
	return b.String()
}

// FormatHypothesisListMD formats a list of hypotheses as a Markdown table.
func FormatHypothesisListMD(hypotheses []*types.HypothesisData, header string) string {
	if len(hypotheses) == 0 {
		return fmt.Sprintf("## %s\n\nNo hypotheses found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(hypotheses))
	fmt.Fprintf(&b, "| ID | Title | Status | Target User | Cycle |\n")
	fmt.Fprintf(&b, "|----|-------|--------|-------------|-------|\n")
	for _, h := range hypotheses {
		cycle := h.CycleID
		if cycle == "" {
			cycle = "—"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", h.ID, h.Title, h.Status, h.TargetUser, cycle)
	}
	return b.String()
}

// FormatExperimentMD formats a single experiment as a Markdown block.
func FormatExperimentMD(e *types.ExperimentData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s\n", e.ID, e.Title)
	fmt.Fprintf(&b, "- **Status:** %s\n", e.Status)
	fmt.Fprintf(&b, "- **Kind:** %s\n", e.Kind)
	fmt.Fprintf(&b, "- **Hypothesis:** %s\n", e.HypothesisID)
	fmt.Fprintf(&b, "- **Question:** %s\n", e.Question)
	fmt.Fprintf(&b, "- **Method:** %s\n", e.Method)
	fmt.Fprintf(&b, "- **Success Signal:** %s\n", e.SuccessSignal)
	fmt.Fprintf(&b, "- **Kill Condition:** %s\n", e.KillCondition)
	if e.CycleID != "" {
		fmt.Fprintf(&b, "- **Cycle:** %s\n", e.CycleID)
	}
	if e.KillTriggered {
		fmt.Fprintf(&b, "- **Kill Triggered:** yes\n")
	}
	if e.Outcome != "" {
		fmt.Fprintf(&b, "- **Outcome:** %s\n", e.Outcome)
	}
	if len(e.Signals) > 0 {
		fmt.Fprintf(&b, "\n**Signals (%d):**\n", len(e.Signals))
		for _, sig := range e.Signals {
			fmt.Fprintf(&b, "- [%s] %s: expected=%s, actual=%s (confidence: %s)\n",
				sig.Type, sig.Metric, sig.Expected, sig.Actual, sig.Confidence)
		}
	}
	if len(e.SpawnedFeatures) > 0 {
		fmt.Fprintf(&b, "- **Spawned Features:** %s\n", strings.Join(e.SpawnedFeatures, ", "))
	}
	if len(e.Labels) > 0 {
		fmt.Fprintf(&b, "- **Labels:** %s\n", strings.Join(e.Labels, ", "))
	}
	return b.String()
}

// FormatExperimentListMD formats a list of experiments as a Markdown table.
func FormatExperimentListMD(experiments []*types.ExperimentData, header string) string {
	if len(experiments) == 0 {
		return fmt.Sprintf("## %s\n\nNo experiments found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(experiments))
	fmt.Fprintf(&b, "| ID | Title | Kind | Status | Hypothesis | Signals |\n")
	fmt.Fprintf(&b, "|----|-------|------|--------|------------|--------|\n")
	for _, e := range experiments {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %d |\n",
			e.ID, e.Title, e.Kind, e.Status, e.HypothesisID, len(e.Signals))
	}
	return b.String()
}

// FormatDiscoveryCycleMD formats a single discovery cycle as a Markdown block.
func FormatDiscoveryCycleMD(c *types.DiscoveryCycleData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s\n", c.ID, c.Title)
	fmt.Fprintf(&b, "- **Status:** %s\n", c.Status)
	fmt.Fprintf(&b, "- **Goal:** %s\n", c.Goal)
	fmt.Fprintf(&b, "- **Period:** %s to %s\n", c.StartDate, c.EndDate)
	if len(c.Hypotheses) > 0 {
		fmt.Fprintf(&b, "- **Hypotheses:** %s\n", strings.Join(c.Hypotheses, ", "))
	}
	if len(c.Experiments) > 0 {
		fmt.Fprintf(&b, "- **Experiments:** %s\n", strings.Join(c.Experiments, ", "))
	}
	if c.Learnings != "" {
		fmt.Fprintf(&b, "- **Learnings:** %s\n", c.Learnings)
	}
	if c.Decision != "" {
		fmt.Fprintf(&b, "- **Decision:** %s\n", c.Decision)
	}
	return b.String()
}

// FormatDiscoveryCycleListMD formats a list of discovery cycles as a Markdown table.
func FormatDiscoveryCycleListMD(cycles []*types.DiscoveryCycleData, header string) string {
	if len(cycles) == 0 {
		return fmt.Sprintf("## %s\n\nNo discovery cycles found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(cycles))
	fmt.Fprintf(&b, "| ID | Title | Status | Period | Decision |\n")
	fmt.Fprintf(&b, "|----|-------|--------|--------|----------|\n")
	for _, c := range cycles {
		decision := c.Decision
		if decision == "" {
			decision = "—"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s–%s | %s |\n",
			c.ID, c.Title, c.Status, c.StartDate, c.EndDate, decision)
	}
	return b.String()
}

// FormatDiscoveryReviewMD formats a single discovery review as a Markdown block.
func FormatDiscoveryReviewMD(r *types.DiscoveryReviewData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s — %s\n", r.ID, r.Title)
	fmt.Fprintf(&b, "- **Cycle:** %s\n", r.CycleID)
	if r.Surprises != "" {
		fmt.Fprintf(&b, "- **Surprises:** %s\n", r.Surprises)
	}
	if r.WrongAbout != "" {
		fmt.Fprintf(&b, "- **Wrong About:** %s\n", r.WrongAbout)
	}
	if r.TransitionReady {
		fmt.Fprintf(&b, "- **Transition Ready:** yes\n")
	}
	if len(r.Items) > 0 {
		fmt.Fprintf(&b, "\n**Decisions (%d):**\n", len(r.Items))
		for _, item := range r.Items {
			fmt.Fprintf(&b, "- %s (%s): **%s** — %s\n",
				item.ItemID, item.ItemType, item.Decision, item.Rationale)
		}
	}
	return b.String()
}

// FormatDiscoveryReviewListMD formats a list of discovery reviews as a Markdown table.
func FormatDiscoveryReviewListMD(reviews []*types.DiscoveryReviewData, header string) string {
	if len(reviews) == 0 {
		return fmt.Sprintf("## %s\n\nNo discovery reviews found.\n", header)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s (%d)\n\n", header, len(reviews))
	fmt.Fprintf(&b, "| ID | Title | Cycle | Items | Transition Ready |\n")
	fmt.Fprintf(&b, "|----|-------|-------|-------|------------------|\n")
	for _, r := range reviews {
		ready := "no"
		if r.TransitionReady {
			ready = "yes"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %s |\n",
			r.ID, r.Title, r.CycleID, len(r.Items), ready)
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
