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

// ProjectPath returns the directory path for a project.
// The result is relative to the workspace root:
// .projects/{slug}
func ProjectPath(projectSlug string) string {
	return filepath.Join(ProjectsDir, projectSlug)
}
