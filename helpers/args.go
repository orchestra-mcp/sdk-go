package helpers

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// GetString extracts a string value from a structpb.Struct by key.
// Returns the empty string if the key is missing or the value is not a string.
func GetString(s *structpb.Struct, key string) string {
	if s == nil {
		return ""
	}
	v, ok := s.Fields[key]
	if !ok || v == nil {
		return ""
	}
	sv, ok := v.Kind.(*structpb.Value_StringValue)
	if !ok {
		return ""
	}
	return sv.StringValue
}

// GetStringOr extracts a string value from a structpb.Struct by key.
// Returns defaultVal if the key is missing, the value is not a string, or the
// string is empty.
func GetStringOr(s *structpb.Struct, key string, defaultVal string) string {
	v := GetString(s, key)
	if v == "" {
		return defaultVal
	}
	return v
}

// GetInt extracts an integer value from a structpb.Struct by key.
// Protobuf Struct stores all numbers as float64, so this truncates to int.
// Returns 0 if the key is missing or the value is not a number.
func GetInt(s *structpb.Struct, key string) int {
	if s == nil {
		return 0
	}
	v, ok := s.Fields[key]
	if !ok || v == nil {
		return 0
	}
	nv, ok := v.Kind.(*structpb.Value_NumberValue)
	if !ok {
		return 0
	}
	return int(nv.NumberValue)
}

// GetFloat64 extracts a float64 value from a structpb.Struct by key.
// Returns 0 if the key is missing or the value is not a number.
func GetFloat64(s *structpb.Struct, key string) float64 {
	if s == nil {
		return 0
	}
	v, ok := s.Fields[key]
	if !ok || v == nil {
		return 0
	}
	nv, ok := v.Kind.(*structpb.Value_NumberValue)
	if !ok {
		return 0
	}
	return nv.NumberValue
}

// GetBool extracts a boolean value from a structpb.Struct by key.
// Returns false if the key is missing or the value is not a bool.
func GetBool(s *structpb.Struct, key string) bool {
	if s == nil {
		return false
	}
	v, ok := s.Fields[key]
	if !ok || v == nil {
		return false
	}
	bv, ok := v.Kind.(*structpb.Value_BoolValue)
	if !ok {
		return false
	}
	return bv.BoolValue
}

// GetStringSlice extracts a string slice from a structpb.Struct by key.
// Each element of the list value is converted to a string; non-string elements
// are silently skipped. Returns nil if the key is missing or the value is not
// a list.
func GetStringSlice(s *structpb.Struct, key string) []string {
	if s == nil {
		return nil
	}
	v, ok := s.Fields[key]
	if !ok || v == nil {
		return nil
	}
	lv, ok := v.Kind.(*structpb.Value_ListValue)
	if !ok || lv.ListValue == nil {
		return nil
	}
	var result []string
	for _, item := range lv.ListValue.Values {
		sv, ok := item.Kind.(*structpb.Value_StringValue)
		if ok {
			result = append(result, sv.StringValue)
		}
	}
	return result
}
