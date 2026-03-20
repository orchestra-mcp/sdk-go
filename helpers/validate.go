package helpers

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// Input length limits.
const (
	MaxProjectIDLen   = 64
	MaxFeatureTitleLen = 500
	MaxNoteBodyLen    = 100 * 1024 // 100 KB
	MaxSearchQueryLen = 1000
	MaxStoragePathLen = 4096
	MaxLabelLen       = 128
	MaxDescriptionLen = 50 * 1024 // 50 KB
)

// ValidateLength checks that s does not exceed maxLen bytes.
// fieldName is used in the error message.
func ValidateLength(s, fieldName string, maxLen int) error {
	if len(s) > maxLen {
		return fmt.Errorf("%s exceeds maximum length (%d > %d)", fieldName, len(s), maxLen)
	}
	return nil
}

// ValidateRequired checks that all named fields exist in the Struct and are
// non-empty. For string values it checks len > 0; for lists, structs, numbers,
// and bools it only checks presence (key exists and value is not null).
func ValidateRequired(args *structpb.Struct, fields ...string) error {
	if args == nil {
		return fmt.Errorf("arguments are required: %s", strings.Join(fields, ", "))
	}
	var missing []string
	for _, f := range fields {
		v, ok := args.Fields[f]
		if !ok || v == nil || v.Kind == nil {
			missing = append(missing, f)
			continue
		}
		// For null values, treat as missing.
		if _, isNull := v.Kind.(*structpb.Value_NullValue); isNull {
			missing = append(missing, f)
			continue
		}
		// For strings, also check non-empty.
		if sv, isStr := v.Kind.(*structpb.Value_StringValue); isStr && sv.StringValue == "" {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateOneOf checks that the given value is one of the allowed values.
// Returns an error if the value is not in the allowed list.
func ValidateOneOf(value string, allowed ...string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("invalid value %q, must be one of: %s", value, strings.Join(allowed, ", "))
}
