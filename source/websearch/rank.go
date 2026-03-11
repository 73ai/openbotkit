package websearch

import (
	"sort"
	"strings"
)

func rankResults(results []Result, query string) []Result {
	type scored struct {
		result Result
		score  int
	}

	// Count URL frequency across backends (dedup bonus).
	urlCount := make(map[string]int)
	for _, r := range results {
		urlCount[normalizeURL(r.URL)]++
	}

	// Deduplicate, keeping first occurrence.
	seen := make(map[string]bool)
	var items []scored
	tokens := queryTokens(query)

	for _, r := range results {
		key := normalizeURL(r.URL)
		if seen[key] {
			continue
		}
		seen[key] = true

		s := urlCount[key] - 1 // frequency bonus
		s += tokenScore(r.Title, tokens, 2)
		s += tokenScore(r.Snippet, tokens, 1)
		if r.Source == "wikipedia" {
			s += 10
		}

		items = append(items, scored{result: r, score: s})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	out := make([]Result, len(items))
	for i, item := range items {
		out[i] = item.result
	}
	return out
}

func queryTokens(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var tokens []string
	for _, w := range words {
		if len(w) > 1 {
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func tokenScore(text string, tokens []string, weight int) int {
	lower := strings.ToLower(text)
	score := 0
	for _, t := range tokens {
		if strings.Contains(lower, t) {
			score += weight
		}
	}
	return score
}
