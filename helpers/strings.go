package helpers

import (
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	leadingTrailing = regexp.MustCompile(`^-+|-+$`)
)

// Slugify converts a string to a URL-safe slug: lowercase, spaces and special
// characters replaced with hyphens, leading/trailing hyphens trimmed.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = leadingTrailing.ReplaceAllString(s, "")
	return s
}

// NowISO returns the current UTC time formatted as ISO 8601.
func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// NewUUID generates a new UUIDv4 string.
func NewUUID() string {
	return uuid.New().String()
}

// NewFeatureID generates a feature ID in the format "FEAT-XXX" where XXX is
// three random uppercase ASCII letters.
func NewFeatureID() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 3)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return "FEAT-" + string(b)
}
