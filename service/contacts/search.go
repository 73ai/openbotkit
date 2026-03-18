package contacts

import (
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/store"
)

func SearchContacts(db *store.DB, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	queryLower := strings.ToLower(query)
	escaped := escapeLike(queryLower)
	prefixPattern := escaped + "%"
	substringPattern := "%" + escaped + "%"

	rows, err := db.Query(
		db.Rebind(`SELECT c.id, c.display_name, c.created_at, c.updated_at,
				a.alias,
				COALESCE(SUM(ci.message_count), 0) as total_messages,
				CASE
					WHEN a.alias_lower = ? THEN 3
					WHEN a.alias_lower LIKE ? THEN 2
					ELSE 1
				END as match_rank
			FROM contacts c
			JOIN contact_aliases a ON a.contact_id = c.id
			LEFT JOIN contact_interactions ci ON ci.contact_id = c.id
			WHERE a.alias_lower LIKE ?
			GROUP BY c.id, a.alias
			ORDER BY match_rank DESC, total_messages DESC, c.display_name
			LIMIT ?`),
		queryLower, prefixPattern, substringPattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}
	defer rows.Close()

	type hit struct {
		id            int64
		alias         string
		totalMessages int
		matchRank     int
	}
	seen := make(map[int64]bool)
	var hits []hit
	for rows.Next() {
		var c Contact
		var h hit
		if err := rows.Scan(&c.ID, &c.DisplayName, &c.CreatedAt, &c.UpdatedAt,
			&h.alias, &h.totalMessages, &h.matchRank); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		h.id = c.ID
		if seen[h.id] {
			continue
		}
		seen[h.id] = true
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, h := range hits {
		full, err := GetContact(db, h.id)
		if err != nil {
			return nil, err
		}
		results = append(results, SearchResult{
			Contact:      *full,
			MatchScore:   h.matchRank*1000 + h.totalMessages,
			MatchedAlias: h.alias,
		})
	}
	return results, nil
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
