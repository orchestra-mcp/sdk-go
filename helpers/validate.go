package helpers

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// ValidateRequired checks that all named fields exist in the Struct and have
// non-empty string values. Returns an error listing all missing or empty fields.
func ValidateRequired(args *structpb.Struct, fields ...string) error {
	if args == nil {
		return fmt.Errorf("arguments are required: %s", strings.Join(fields, ", "))
	}
	var missing []string
	for _, f := range fields {
		v := GetString(args, f)
		if v == "" {
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
