package target

import "github.com/fun7257/arrow"

// Page is a paginated collection payload.
type Page[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Size  int `json:"size"`
}

// WritePage writes a paginated JSON response.
func WritePage[T any](c *arrow.Context, status int, page Page[T]) error {
	return WriteJSON(c, status, page)
}