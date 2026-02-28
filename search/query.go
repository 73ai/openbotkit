package search

// FetchQuery defines a broad fetch filter - just from address and date.
type FetchQuery struct {
	From  string // e.g. "anthropic.com"
	After string // date string "2025/07/17"
}

// Build composes a Gmail search query string.
func (q FetchQuery) Build() string {
	s := "from:" + q.From
	if q.After != "" {
		s += " after:" + q.After
	}
	return s
}

