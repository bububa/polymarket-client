package shared

// Page wraps a paginated API response.
type Page[T any] struct {
	// Limit is the requested page size.
	Limit Int `json:"limit"`
	// Count is the number of results on this page.
	Count Int `json:"count"`
	// NextCursor is the pagination cursor for the next page, empty when exhausted.
	NextCursor string `json:"next_cursor"`
	// Data contains the page results.
	Data []T `json:"data"`
}
