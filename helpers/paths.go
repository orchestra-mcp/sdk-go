package helpers

import "path/filepath"

const (
	// ProjectsDir is the top-level directory where all project data is stored.
	ProjectsDir = ".projects"

	// FeaturesDir is the subdirectory within a project that holds feature files.
	FeaturesDir = "features"

	// PlansDir is the subdirectory within a project that holds plan files.
	PlansDir = "plans"

	// RequestsDir is the subdirectory within a project that holds request files.
	RequestsDir = "requests"

	// PersonsDir is the subdirectory within a project that holds person files.
	PersonsDir = "persons"

	// AssignmentRulesDir is the subdirectory within a project that holds assignment rule files.
	AssignmentRulesDir = "assignment-rules"

	// HypothesesDir is the subdirectory within a project that holds hypothesis files.
	HypothesesDir = "hypotheses"

	// ExperimentsDir is the subdirectory within a project that holds experiment files.
	ExperimentsDir = "experiments"

	// DiscoveryCyclesDir is the subdirectory within a project that holds discovery cycle files.
	DiscoveryCyclesDir = "discovery-cycles"

	// DiscoveryReviewsDir is the subdirectory within a project that holds discovery review files.
	DiscoveryReviewsDir = "discovery-reviews"

	// DelegationsDir is the subdirectory within a project that holds delegation files.
	DelegationsDir = "delegations"

	// ConfigFile is the name of the project configuration file.
	ConfigFile = "project.json"
)

// FeaturePath returns the file path for a feature within a project.
// The result is relative to the workspace root:
// .projects/{slug}/features/{id}.md
func FeaturePath(projectSlug, featureID string) string {
	return filepath.Join(ProjectsDir, projectSlug, FeaturesDir, featureID+".md")
}

// PlanPath returns the file path for a plan within a project.
// The result is relative to the workspace root:
// .projects/{slug}/plans/{id}.md
func PlanPath(projectSlug, planID string) string {
	return filepath.Join(ProjectsDir, projectSlug, PlansDir, planID+".md")
}

// RequestPath returns the file path for a request within a project.
// The result is relative to the workspace root:
// .projects/{slug}/requests/{id}.md
func RequestPath(projectSlug, requestID string) string {
	return filepath.Join(ProjectsDir, projectSlug, RequestsDir, requestID+".md")
}

// PersonPath returns the file path for a person within a project.
// The result is relative to the workspace root:
// .projects/{slug}/persons/{id}.md
func PersonPath(projectSlug, personID string) string {
	return filepath.Join(ProjectsDir, projectSlug, PersonsDir, personID+".md")
}

// AssignmentRulePath returns the file path for an assignment rule within a project.
// The result is relative to the workspace root:
// .projects/{slug}/assignment-rules/{id}.md
func AssignmentRulePath(projectSlug, ruleID string) string {
	return filepath.Join(ProjectsDir, projectSlug, AssignmentRulesDir, ruleID+".md")
}

// HypothesisPath returns the file path for a hypothesis within a project.
// The result is relative to the workspace root:
// .projects/{slug}/hypotheses/{id}.md
func HypothesisPath(projectSlug, hypoID string) string {
	return filepath.Join(ProjectsDir, projectSlug, HypothesesDir, hypoID+".md")
}

// ExperimentPath returns the file path for an experiment within a project.
// The result is relative to the workspace root:
// .projects/{slug}/experiments/{id}.md
func ExperimentPath(projectSlug, exprID string) string {
	return filepath.Join(ProjectsDir, projectSlug, ExperimentsDir, exprID+".md")
}

// DiscoveryCyclePath returns the file path for a discovery cycle within a project.
// The result is relative to the workspace root:
// .projects/{slug}/discovery-cycles/{id}.md
func DiscoveryCyclePath(projectSlug, cycleID string) string {
	return filepath.Join(ProjectsDir, projectSlug, DiscoveryCyclesDir, cycleID+".md")
}

// DiscoveryReviewPath returns the file path for a discovery review within a project.
// The result is relative to the workspace root:
// .projects/{slug}/discovery-reviews/{id}.md
func DiscoveryReviewPath(projectSlug, reviewID string) string {
	return filepath.Join(ProjectsDir, projectSlug, DiscoveryReviewsDir, reviewID+".md")
}

// DelegationPath returns the file path for a delegation within a project.
// The result is relative to the workspace root:
// .projects/{slug}/delegations/{id}.md
func DelegationPath(projectSlug, delegationID string) string {
	return filepath.Join(ProjectsDir, projectSlug, DelegationsDir, delegationID+".md")
}

// ProjectPath returns the directory path for a project.
// The result is relative to the workspace root:
// .projects/{slug}
func ProjectPath(projectSlug string) string {
	return filepath.Join(ProjectsDir, projectSlug)
}
