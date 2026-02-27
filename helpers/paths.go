package helpers

import "path/filepath"

const (
	// ProjectsDir is the top-level directory where all project data is stored.
	ProjectsDir = ".projects"

	// FeaturesDir is the subdirectory within a project that holds feature files.
	FeaturesDir = "features"

	// ConfigFile is the name of the project configuration file.
	ConfigFile = "project.json"
)

// FeaturePath returns the file path for a feature within a project.
// The result is relative to the workspace root:
// .projects/{slug}/features/{id}.md
func FeaturePath(projectSlug, featureID string) string {
	return filepath.Join(ProjectsDir, projectSlug, FeaturesDir, featureID+".md")
}

// ProjectPath returns the directory path for a project.
// The result is relative to the workspace root:
// .projects/{slug}
func ProjectPath(projectSlug string) string {
	return filepath.Join(ProjectsDir, projectSlug)
}
