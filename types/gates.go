package types

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// GateID identifies a specific workflow gate.
type GateID string

const (
	GateCodeComplete GateID = "code_complete"
	GateTestComplete GateID = "test_complete"
	GateDocsComplete GateID = "docs_complete"
)

// GateRequirement defines what evidence is needed to pass through a gated
// workflow transition. Evidence is validated by checking for a required markdown
// section (## header) that must reference file paths.
type GateRequirement struct {
	ID                GateID
	From              FeatureStatus
	To                FeatureStatus
	Name              string
	RequiredSection   string        // section name (without "## " prefix)
	MinFilePaths      int           // minimum distinct file paths required
	FilePatterns      []string      // expected file patterns for validation (empty = any)
	DocsFolder        string        // if set, file paths must be under this folder
	SkippableForKinds []FeatureKind // gate can be auto-passed for these kinds
}

// filePathPattern matches common file path patterns:
// - paths with extensions: foo/bar.go, src/main.rs, docs/README.md
// - paths with slashes: libs/sdk-go/types/gates.go
// - relative paths: ./src/main.go, ../config.yaml
var filePathPattern = regexp.MustCompile(`(?:^|[\s` + "`" + `\-*])([.\w][\w./\-]*\.\w{1,10})(?:\s|$|[,:;()\]` + "`" + `])`)

// Validate checks whether the provided evidence satisfies this gate's
// requirements. Returns nil if passed, or an error describing what is missing.
func (g *GateRequirement) Validate(evidence string) error {
	trimmed := strings.TrimSpace(evidence)
	if trimmed == "" {
		return fmt.Errorf("gate %q requires evidence with a ## %s section listing file paths", g.Name, g.RequiredSection)
	}

	// Parse sections from evidence.
	sections := parseSections(trimmed)
	content, found := sections[strings.ToLower(g.RequiredSection)]
	if !found {
		return fmt.Errorf("evidence missing required section: ## %s", g.RequiredSection)
	}
	if len(strings.TrimSpace(content)) < 10 {
		return fmt.Errorf("section ## %s has insufficient content (minimum 10 characters)", g.RequiredSection)
	}

	// Count file paths.
	minPaths := g.MinFilePaths
	if minPaths <= 0 {
		minPaths = 1
	}
	paths := ExtractFilePaths(content)
	if len(paths) < minPaths {
		return fmt.Errorf("section ## %s must reference at least %d file path(s) (found %d). "+
			"List the actual files changed, e.g.: src/main.go, tests/auth_test.go",
			g.RequiredSection, minPaths, len(paths))
	}

	return nil
}

// CheckFileTypes validates that extracted file paths match the expected patterns
// for this gate. Returns (true, nil) if at least one file matches or no patterns
// are configured. Returns (false, expectedPatterns) if none match.
func (g *GateRequirement) CheckFileTypes(evidence string) (ok bool, expected []string) {
	if len(g.FilePatterns) == 0 && g.DocsFolder == "" {
		return true, nil
	}

	sections := parseSections(strings.TrimSpace(evidence))
	content := sections[strings.ToLower(g.RequiredSection)]
	paths := ExtractFilePaths(content)

	if len(paths) == 0 {
		return false, g.FilePatterns
	}

	// Check docs folder constraint.
	if g.DocsFolder != "" {
		for _, p := range paths {
			if strings.HasPrefix(p, g.DocsFolder+"/") || strings.HasPrefix(p, g.DocsFolder+"\\") {
				if strings.HasSuffix(strings.ToLower(p), ".md") {
					return true, nil
				}
			}
		}
		return false, []string{g.DocsFolder + "/*.md"}
	}

	// Check file patterns.
	for _, p := range paths {
		for _, pattern := range g.FilePatterns {
			if matchesFilePattern(p, pattern) {
				return true, nil
			}
		}
	}
	return false, g.FilePatterns
}

// IsSkippableFor reports whether this gate can be auto-passed for the given kind.
func (g *GateRequirement) IsSkippableFor(kind FeatureKind) bool {
	for _, k := range g.SkippableForKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// matchesFilePattern checks if a file path matches a pattern suffix.
// e.g., "_test.go" matches "src/auth_test.go"
func matchesFilePattern(path, pattern string) bool {
	lower := strings.ToLower(path)
	lowerPattern := strings.ToLower(pattern)

	// Glob-style pattern (e.g., "docs/*.md")
	if strings.Contains(lowerPattern, "*") {
		matched, _ := filepath.Match(lowerPattern, lower)
		if matched {
			return true
		}
		// Also try just the filename
		matched, _ = filepath.Match(lowerPattern, filepath.Base(lower))
		return matched
	}

	// Suffix match (e.g., "_test.go" matches "src/auth_test.go")
	return strings.HasSuffix(lower, lowerPattern)
}

// ExtractFilePaths extracts unique file paths from text.
func ExtractFilePaths(text string) []string {
	matches := filePathPattern.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	var paths []string
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			paths = append(paths, m[1])
		}
	}
	return paths
}

// CountDistinctFilePaths counts the number of unique file paths matched in text.
func CountDistinctFilePaths(text string) int {
	return len(ExtractFilePaths(text))
}

// parseSections extracts markdown sections from evidence text. Returns a map
// of lowercase section name → section content.
func parseSections(text string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(text, "\n")

	var currentSection string
	var currentContent strings.Builder

	for _, line := range lines {
		trimLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimLine, "## ") {
			if currentSection != "" {
				sections[currentSection] = currentContent.String()
			}
			currentSection = strings.ToLower(strings.TrimSpace(trimLine[3:]))
			currentContent.Reset()
		} else if currentSection != "" {
			if currentContent.Len() > 0 || trimLine != "" {
				currentContent.WriteString(line)
				currentContent.WriteString("\n")
			}
		}
	}

	if currentSection != "" {
		sections[currentSection] = currentContent.String()
	}

	return sections
}

// GateRequirements maps gated transitions to their requirements.
// Only transitions in this map require evidence; all others are free.
var GateRequirements = map[FeatureStatus]map[FeatureStatus]*GateRequirement{
	StatusInProgress: {
		StatusInTesting: {
			ID:              GateCodeComplete,
			From:            StatusInProgress,
			To:              StatusInTesting,
			Name:            "Code Complete",
			RequiredSection: "Changes",
			MinFilePaths:    1,
		},
	},
	StatusInTesting: {
		StatusInDocs: {
			ID:              GateTestComplete,
			From:            StatusInTesting,
			To:              StatusInDocs,
			Name:            "Test Complete",
			RequiredSection: "Results",
			MinFilePaths:    1,
			FilePatterns:    []string{"_test.go", ".test.ts", ".test.tsx", ".spec.ts", ".spec.tsx", "_test.rs", ".test.js", ".spec.js", ".test.py", "_test.py"},
		},
		StatusInReview: {
			ID:              GateTestComplete,
			From:            StatusInTesting,
			To:              StatusInReview,
			Name:            "Test Complete (skip docs)",
			RequiredSection: "Results",
			MinFilePaths:    1,
			FilePatterns:    []string{"_test.go", ".test.ts", ".test.tsx", ".spec.ts", ".spec.tsx", "_test.rs", ".test.js", ".spec.js", ".test.py", "_test.py"},
			SkippableForKinds: []FeatureKind{KindBug, KindHotfix, KindTestcase},
		},
	},
	StatusInDocs: {
		StatusInReview: {
			ID:              GateDocsComplete,
			From:            StatusInDocs,
			To:              StatusInReview,
			Name:            "Docs Complete",
			RequiredSection: "Docs",
			MinFilePaths:    1,
			DocsFolder:      "docs",
			SkippableForKinds: []FeatureKind{KindBug, KindHotfix, KindTestcase},
		},
	},
}

// GetGate returns the gate requirement for a given transition, or nil if the
// transition is not gated (i.e., it is a free transition).
func GetGate(from, to FeatureStatus) *GateRequirement {
	toMap, ok := GateRequirements[from]
	if !ok {
		return nil
	}
	return toMap[to]
}

// IsGated reports whether the transition from one status to another requires
// evidence to pass through a gate.
func IsGated(from, to FeatureStatus) bool {
	return GetGate(from, to) != nil
}
