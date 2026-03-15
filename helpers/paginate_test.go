package helpers

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestParsePagination_Defaults(t *testing.T) {
	s, _ := structpb.NewStruct(map[string]any{})
	pg := ParsePagination(s)
	if pg.Limit != DefaultPageLimit {
		t.Errorf("Limit = %d, want %d", pg.Limit, DefaultPageLimit)
	}
	if pg.Offset != 0 {
		t.Errorf("Offset = %d, want 0", pg.Offset)
	}
}

func TestParsePagination_CustomValues(t *testing.T) {
	s, _ := structpb.NewStruct(map[string]any{
		"limit":  float64(25),
		"offset": float64(10),
	})
	pg := ParsePagination(s)
	if pg.Limit != 25 {
		t.Errorf("Limit = %d, want 25", pg.Limit)
	}
	if pg.Offset != 10 {
		t.Errorf("Offset = %d, want 10", pg.Offset)
	}
}

func TestParsePagination_ClampToMax(t *testing.T) {
	s, _ := structpb.NewStruct(map[string]any{
		"limit": float64(999),
	})
	pg := ParsePagination(s)
	if pg.Limit != MaxPageLimit {
		t.Errorf("Limit = %d, want %d (clamped)", pg.Limit, MaxPageLimit)
	}
}

func TestParsePagination_NegativeValues(t *testing.T) {
	s, _ := structpb.NewStruct(map[string]any{
		"limit":  float64(-1),
		"offset": float64(-5),
	})
	pg := ParsePagination(s)
	if pg.Limit != DefaultPageLimit {
		t.Errorf("Limit = %d, want %d (negative should default)", pg.Limit, DefaultPageLimit)
	}
	if pg.Offset != 0 {
		t.Errorf("Offset = %d, want 0 (negative should clamp to 0)", pg.Offset)
	}
}

func TestParsePagination_NilArgs(t *testing.T) {
	pg := ParsePagination(nil)
	if pg.Limit != DefaultPageLimit {
		t.Errorf("Limit = %d, want %d", pg.Limit, DefaultPageLimit)
	}
	if pg.Offset != 0 {
		t.Errorf("Offset = %d, want 0", pg.Offset)
	}
}

func TestPaginateSlice_NormalPage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := PaginateSlice(items, PaginationParams{Limit: 3, Offset: 2})
	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
	if result[0] != 3 || result[1] != 4 || result[2] != 5 {
		t.Errorf("expected [3,4,5], got %v", result)
	}
}

func TestPaginateSlice_LastPage(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	result := PaginateSlice(items, PaginationParams{Limit: 3, Offset: 3})
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0] != 4 || result[1] != 5 {
		t.Errorf("expected [4,5], got %v", result)
	}
}

func TestPaginateSlice_OffsetBeyondEnd(t *testing.T) {
	items := []int{1, 2, 3}
	result := PaginateSlice(items, PaginationParams{Limit: 10, Offset: 100})
	if result != nil {
		t.Errorf("expected nil for offset beyond end, got %v", result)
	}
}

func TestPaginateSlice_EmptySlice(t *testing.T) {
	var items []int
	result := PaginateSlice(items, PaginationParams{Limit: 10, Offset: 0})
	if result != nil {
		t.Errorf("expected nil for empty slice, got %v", result)
	}
}

func TestPaginateSlice_FullPage(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := PaginateSlice(items, PaginationParams{Limit: 50, Offset: 0})
	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
}
