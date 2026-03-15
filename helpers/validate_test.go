package helpers

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestValidateLength_WithinLimit(t *testing.T) {
	err := ValidateLength("short", "field", 100)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateLength_ExactLimit(t *testing.T) {
	s := strings.Repeat("a", 64)
	err := ValidateLength(s, "project_id", 64)
	if err != nil {
		t.Fatalf("expected no error at exact limit, got: %v", err)
	}
}

func TestValidateLength_OverLimit(t *testing.T) {
	s := strings.Repeat("a", 65)
	err := ValidateLength(s, "project_id", 64)
	if err == nil {
		t.Fatal("expected error for over-limit input")
	}
	if !strings.Contains(err.Error(), "project_id") {
		t.Errorf("error should mention field name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "65") {
		t.Errorf("error should mention actual length, got: %v", err)
	}
}

func TestValidateLength_EmptyString(t *testing.T) {
	err := ValidateLength("", "field", 100)
	if err != nil {
		t.Fatalf("empty string should pass, got: %v", err)
	}
}

func TestValidateRequired_AllPresent(t *testing.T) {
	s, _ := structpb.NewStruct(map[string]any{
		"name": "test",
		"id":   "123",
	})
	err := ValidateRequired(s, "name", "id")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateRequired_MissingFields(t *testing.T) {
	s, _ := structpb.NewStruct(map[string]any{
		"name": "test",
	})
	err := ValidateRequired(s, "name", "id", "email")
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	if !strings.Contains(err.Error(), "id") || !strings.Contains(err.Error(), "email") {
		t.Errorf("error should list missing fields, got: %v", err)
	}
}

func TestValidateRequired_NilStruct(t *testing.T) {
	err := ValidateRequired(nil, "name")
	if err == nil {
		t.Fatal("expected error for nil struct")
	}
}

func TestValidateOneOf_Valid(t *testing.T) {
	err := ValidateOneOf("P1", "P0", "P1", "P2", "P3")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateOneOf_Invalid(t *testing.T) {
	err := ValidateOneOf("P5", "P0", "P1", "P2", "P3")
	if err == nil {
		t.Fatal("expected error for invalid value")
	}
	if !strings.Contains(err.Error(), "P5") {
		t.Errorf("error should mention invalid value, got: %v", err)
	}
}

func TestValidateLengthConstants(t *testing.T) {
	// Verify constants are set to expected values.
	if MaxProjectIDLen != 64 {
		t.Errorf("MaxProjectIDLen = %d, want 64", MaxProjectIDLen)
	}
	if MaxFeatureTitleLen != 500 {
		t.Errorf("MaxFeatureTitleLen = %d, want 500", MaxFeatureTitleLen)
	}
	if MaxNoteBodyLen != 100*1024 {
		t.Errorf("MaxNoteBodyLen = %d, want %d", MaxNoteBodyLen, 100*1024)
	}
	if MaxSearchQueryLen != 1000 {
		t.Errorf("MaxSearchQueryLen = %d, want 1000", MaxSearchQueryLen)
	}
	if MaxStoragePathLen != 4096 {
		t.Errorf("MaxStoragePathLen = %d, want 4096", MaxStoragePathLen)
	}
}
