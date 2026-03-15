package helpers

import "google.golang.org/protobuf/types/known/structpb"

const (
	DefaultPageLimit = 50
	MaxPageLimit     = 200
)

// PaginationParams holds parsed limit/offset values.
type PaginationParams struct {
	Limit  int
	Offset int
}

// ParsePagination extracts limit and offset from a structpb.Struct, applying
// defaults and clamping to MaxPageLimit.
func ParsePagination(args *structpb.Struct) PaginationParams {
	limit := GetInt(args, "limit")
	offset := GetInt(args, "offset")

	if limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return PaginationParams{Limit: limit, Offset: offset}
}

// PaginateSlice applies offset and limit to a slice, returning the paginated
// sub-slice. It is safe to call with out-of-bounds offset (returns empty).
// T can be any type.
func PaginateSlice[T any](items []T, p PaginationParams) []T {
	total := len(items)
	if p.Offset >= total {
		return nil
	}
	end := p.Offset + p.Limit
	if end > total {
		end = total
	}
	return items[p.Offset:end]
}
