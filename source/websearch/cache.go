package websearch

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/priyanshujain/openbotkit/store"
)

func cacheKey(query, category, backend, region, timeLimit string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s", query, category, backend, region, timeLimit)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func getSearchCache(db *store.DB, key string, ttl time.Duration) (*SearchResult, bool) {
	if db == nil {
		return nil, false
	}

	var query, resultsJSON string
	var createdAt time.Time
	q := db.Rebind("SELECT query, results, created_at FROM search_cache WHERE cache_key = ?")
	err := db.QueryRow(q, key).Scan(&query, &resultsJSON, &createdAt)
	if err != nil {
		return nil, false
	}

	if time.Since(createdAt) > ttl {
		delQ := db.Rebind("DELETE FROM search_cache WHERE cache_key = ?")
		db.Exec(delQ, key)
		return nil, false
	}

	var results []Result
	if err := json.Unmarshal([]byte(resultsJSON), &results); err != nil {
		return nil, false
	}

	return &SearchResult{
		Query:   query,
		Results: results,
		Metadata: SearchMetadata{
			Backends:     []string{"cache"},
			TotalResults: len(results),
			Cached:       true,
		},
	}, true
}

func putSearchCache(db *store.DB, key, query, category string, results []Result) {
	if db == nil {
		return
	}

	data, err := json.Marshal(results)
	if err != nil {
		return
	}

	q := db.Rebind(`INSERT INTO search_cache (cache_key, query, category, results) VALUES (?, ?, ?, ?)
		ON CONFLICT(cache_key) DO UPDATE SET results = excluded.results, created_at = CURRENT_TIMESTAMP`)
	db.Exec(q, key, query, category, string(data))
}

func getFetchCache(db *store.DB, url, format string, ttl time.Duration) (*FetchResult, bool) {
	if db == nil {
		return nil, false
	}

	var title, content, storedFormat string
	var statusCode int
	var fetchedAt time.Time
	q := db.Rebind("SELECT title, content, format, status_code, fetched_at FROM fetch_cache WHERE url = ?")
	err := db.QueryRow(q, url).Scan(&title, &content, &storedFormat, &statusCode, &fetchedAt)
	if err != nil {
		return nil, false
	}

	if time.Since(fetchedAt) > ttl {
		delQ := db.Rebind("DELETE FROM fetch_cache WHERE url = ?")
		db.Exec(delQ, url)
		return nil, false
	}

	if storedFormat != format {
		return nil, false
	}

	return &FetchResult{
		URL:        url,
		Title:      title,
		Content:    content,
		StatusCode: statusCode,
		Cached:     true,
	}, true
}

func putFetchCache(db *store.DB, result *FetchResult, format string) {
	if db == nil || result == nil {
		return
	}

	q := db.Rebind(`INSERT INTO fetch_cache (url, title, content, format, status_code) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET title = excluded.title, content = excluded.content,
		format = excluded.format, status_code = excluded.status_code, fetched_at = CURRENT_TIMESTAMP`)
	db.Exec(q, result.URL, result.Title, result.Content, format, result.StatusCode)
}

