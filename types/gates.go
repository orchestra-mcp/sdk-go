package types

import (
	"fmt"
	"regexp"
	"strings"
)

// GateID identifies a specific workflow gate.
type GateID string

const (
	GateImplementation  GateID = "implementation"
	GateTesting         GateID = "testing"
	GateDocumentation   GateID = "documentation"
	GateReviewSelfCheck GateID = "review_self_check"
)

// GateRequirement defines what evidence is needed to pass through a gated
// workflow transition. Evidence is validated by checking for required markdown
// sections (## headers) with minimum content, and optionally requiring that
// specific sections reference actual file paths.
type GateRequirement struct {
	ID                GateID
	From              FeatureStatus
	To                FeatureStatus
	Name              string
	RequiredSections  []string      // section names (without "## " prefix)
	FilePathSections  []string      // sections that MUST contain at least one file path
	MinSectionLen     int           // minimum characters of content per section
	MinTotalLen       int           // minimum total characters across all evidence (0 = no check)
	MinFilePaths      int           // minimum distinct file paths required in FilePathSections (0 = just 1)
	SkippableForKinds []FeatureKind // gate can be auto-passed for these kinds
	Checklist         string        // markdown checklist returned to the agent on rejection
}

// filePathPattern matches common file path patterns:
// - paths with extensions: foo/bar.go, src/main.rs, docs/README.md
// - paths with slashes: libs/sdk-go/types/gates.go
// - relative paths: ./src/main.go, ../config.yaml
var filePathPattern = regexp.MustCompile(`(?:^|[\s` + "`" + `\-*])([.\w][\w./\-]*\.\w{1,10})(?:\s|$|[,:;()\]` + "`" + `])`)


// IsSkippableFor reports whether this gate can be auto-passed for the given kind.
func (g *GateRequirement) IsSkippableFor(kind FeatureKind) bool {
	for _, k := range g.SkippableForKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// Validate checks whether the provided evidence satisfies this gate's
// requirements. Returns nil if the evidence passes, or an error describing
// what is missing. Validation is agnostic — it checks structure (sections
// present with substance), not domain-specific content.
func (g *GateRequirement) Validate(evidence string) error {
	trimmed := strings.TrimSpace(evidence)
	if trimmed == "" {
		return fmt.Errorf("gate %q requires evidence — call get_gate_requirements to see what is needed", g.Name)
	}
	if len(trimmed) < 30 {
		return fmt.Errorf("evidence too short: gate %q requires at least 30 characters (got %d)", g.Name, len(trimmed))
	}

	// Check minimum total evidence length if configured.
	if g.MinTotalLen > 0 && len(trimmed) < g.MinTotalLen {
		return fmt.Errorf("evidence too short: gate %q requires at least %d characters total (got %d). "+
			"Provide detailed, substantive evidence that reflects real work performed",
			g.Name, g.MinTotalLen, len(trimmed))
	}

	minLen := g.MinSectionLen
	if minLen == 0 {
		minLen = 10
	}

	// Parse sections from evidence.
	sections := parseSections(trimmed)

	var missing []string
	var tooShort []string
	for _, required := range g.RequiredSections {
		content, found := sections[strings.ToLower(required)]
		if !found {
			missing = append(missing, required)
			continue
		}
		if len(strings.TrimSpace(content)) < minLen {
			tooShort = append(tooShort, required)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("evidence missing required sections: %s. Each section must start with '## SectionName'",
			strings.Join(missing, ", "))
	}
	if len(tooShort) > 0 {
		return fmt.Errorf("sections with insufficient content (minimum %d chars each): %s",
			minLen, strings.Join(tooShort, ", "))
	}

	// Validate that file-path-required sections contain minimum distinct file paths.
	minPaths := g.MinFilePaths
	if minPaths <= 0 {
		minPaths = 1
	}
	var tooFewFiles []string
	for _, reqSection := range g.FilePathSections {
		content, found := sections[strings.ToLower(reqSection)]
		if !found {
			continue // already caught by missing sections check above
		}
		paths := CountDistinctFilePaths(content)
		if paths < minPaths {
			tooFewFiles = append(tooFewFiles, fmt.Sprintf("%s (found %d, need %d)", reqSection, paths, minPaths))
		}
	}
	if len(tooFewFiles) > 0 {
		return fmt.Errorf("sections must reference at least %d distinct file path(s) (e.g., src/main.go, docs/README.md): %s. "+
			"Evidence must reflect real file changes, not just prose descriptions",
			minPaths, strings.Join(tooFewFiles, ", "))
	}

	return nil
}

// CountDistinctFilePaths counts the number of unique file paths matched in text.
func CountDistinctFilePaths(text string) int {
	matches := filePathPattern.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 {
			seen[m[1]] = true
		}
	}
	return len(seen)
}

// parseSections extracts markdown sections from evidence text. Returns a map
// of lowercase section name → section content (text between this header and
// the next header or end of string).
func parseSections(text string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(text, "\n")

	var currentSection string
	var currentContent strings.Builder

	for _, line := range lines {
		trimLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimLine, "## ") {
			// Save previous section if any.
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

	// Save last section.
	if currentSection != "" {
		sections[currentSection] = currentContent.String()
	}

	return sections
}

// GateRequirements maps gated transitions to their requirements. Only
// transitions that appear in this map require evidence; all others are free.
var GateRequirements = map[FeatureStatus]map[FeatureStatus]*GateRequirement{
	StatusInProgress: {
		StatusReadyForTesting: {
			ID:               GateImplementation,
			From:             StatusInProgress,
			To:               StatusReadyForTesting,
			Name:             "Implementation Complete",
			RequiredSections: []string{"Summary", "Changes", "Verification"},
			FilePathSections: []string{"Changes"},
			MinSectionLen:    20,
			MinTotalLen:      100,
			MinFilePaths:     1,
			Checklist: `## Gate 1: Implementation Complete

Before advancing from **in-progress** to **ready-for-testing**, provide evidence with these sections:

### Required Sections
- **## Summary** — What was implemented? Describe the changes at a high level.
- **## Changes** — What files were modified or created? **Must include actual file paths** (e.g., ` + "`" + `libs/sdk-go/types/gates.go` + "`" + `).
- **## Verification** — How can someone verify this works? Steps to test.

### Evidence Format
` + "```" + `
evidence: "## Summary\n<describe what was built>\n\n## Changes\n- libs/foo/bar.go (added validation)\n- libs/baz/qux.go (new file)\n\n## Verification\n<describe how to test>"
` + "```" + `

Each section must have at least 20 characters of content, and total evidence must be at least 100 characters. The **Changes** section must reference actual file paths.

**Anti-bypass:** Do NOT use sleep/wait to bypass gate cooldowns. Do real work between gates. Evidence is checked for substance — templated or copied evidence will be rejected.`,
		},
	},
	StatusInTesting: {
		StatusReadyForDocs: {
			ID:               GateTesting,
			From:             StatusInTesting,
			To:               StatusReadyForDocs,
			Name:             "Testing Complete",
			RequiredSections: []string{"Summary", "Results", "Coverage"},
			MinSectionLen:    20,
			MinTotalLen:      100,
			Checklist: `## Gate 2: Testing Complete

Before advancing from **in-testing** to **ready-for-docs**, provide evidence with these sections:

### Required Sections
- **## Summary** — What was tested? Describe the testing scope.
- **## Results** — What were the outcomes? Pass/fail, issues found.
- **## Coverage** — What is the test coverage? Areas covered and any gaps.

### Evidence Format
` + "```" + `
evidence: "## Summary\n<describe testing scope>\n\n## Results\n<describe outcomes>\n\n## Coverage\n<describe coverage>"
` + "```" + `

Each section must have at least 20 characters of content, and total evidence must be at least 100 characters.

**Anti-bypass:** Do NOT use sleep/wait to bypass gate cooldowns. Do real work between gates.`,
		},
	},
	StatusInDocs: {
		StatusDocumented: {
			ID:                GateDocumentation,
			From:              StatusInDocs,
			To:                StatusDocumented,
			Name:              "Documentation Complete",
			RequiredSections:  []string{"Summary", "Location"},
			FilePathSections:  []string{"Location"},
			MinSectionLen:     20,
			MinTotalLen:       80,
			MinFilePaths:      1,
			SkippableForKinds: []FeatureKind{KindBug, KindHotfix},
			Checklist: `## Gate 3: Documentation Complete

Before advancing from **in-docs** to **documented**, provide evidence with these sections:

### Required Sections
- **## Summary** — What was documented? Describe the scope.
- **## Location** — Where do the docs live? **Must include actual file paths** to the documentation files (e.g., ` + "`" + `docs/feature-x.md` + "`" + `).

### Evidence Format
` + "```" + `
evidence: "## Summary\n<describe what was documented>\n\n## Location\n- docs/feature-x.md (new)\n- CHANGELOG.md (updated)"
` + "```" + `

Each section must have at least 20 characters of content, and total evidence must be at least 80 characters. The **Location** section must reference actual file paths.

**Anti-bypass:** Do NOT use sleep/wait to bypass gate cooldowns. Do real work between gates.`,
		},
	},
}

// ReviewGate defines the self-review evidence required when requesting a
// human review via request_review. This is separate from GateRequirements
// because it applies to the request_review tool, not advance_feature.
var ReviewGate = &GateRequirement{
	ID:               GateReviewSelfCheck,
	From:             StatusDocumented,
	To:               StatusInReview,
	Name:             "Self-Review",
	RequiredSections: []string{"Summary", "Quality", "Checklist"},
	FilePathSections: []string{"Checklist"},
	MinSectionLen:    20,
	MinTotalLen:      120,
	MinFilePaths:     1,
	Checklist: `## Gate 4: Self-Review (before human review)

Before requesting a human review, provide your self-review evidence with these sections:

### Required Sections
- **## Summary** — What is this feature? Brief description of the deliverable.
- **## Quality** — Your assessment of the work quality. Any concerns?
- **## Checklist** — What was completed? **Must reference actual files** that were created or modified.

### Evidence Format
` + "```" + `
evidence: "## Summary\n<what this feature does>\n\n## Quality\n<your quality assessment>\n\n## Checklist\n- [x] libs/foo/bar.go — added validation\n- [x] docs/feature.md — documentation"
` + "```" + `

Each section must have at least 20 characters of content, and total evidence must be at least 120 characters. The **Checklist** section must reference actual file paths.

**Anti-bypass:** Do NOT use sleep/wait to bypass gate cooldowns. Do real work between gates.

After this, the MCP will instruct you to ask the user for final approval.`,
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
