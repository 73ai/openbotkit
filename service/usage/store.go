package usage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// UsageRecord represents a single LLM API call's token usage.
type UsageRecord struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	Channel          string `json:"channel"`
	SessionID        string `json:"session_id"`
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	CacheReadTokens  int    `json:"cache_read_tokens"`
	CacheWriteTokens int    `json:"cache_write_tokens"`
	CreatedAt        string `json:"created_at"`
}

// AggregatedUsage groups token counts by date and model.
type AggregatedUsage struct {
	Date             string
	Model            string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	CallCount        int64
}

// QueryOpts controls filtering for aggregated queries.
type QueryOpts struct {
	Since    *time.Time
	Until    *time.Time
	Model    string
	Provider string
	Channel  string
	GroupBy  string // "daily" (default) or "monthly"
}

// Migrate creates the parent directory for the JSONL file.
func Migrate(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0700)
}

// Record appends a usage record as a JSON line to the file.
func Record(path string, rec UsageRecord) error {
	if rec.CreatedAt == "" {
		rec.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open usage file: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(rec); err != nil {
		return fmt.Errorf("write usage record: %w", err)
	}
	return nil
}

// Query reads the JSONL file, filters by opts, and returns aggregated results.
func Query(path string, opts QueryOpts) ([]AggregatedUsage, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open usage file: %w", err)
	}
	defer f.Close()

	type groupKey struct {
		Date  string
		Model string
	}
	groups := make(map[groupKey]*AggregatedUsage)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec UsageRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}

		createdAt, err := time.Parse(time.RFC3339, rec.CreatedAt)
		if err != nil {
			continue
		}

		if opts.Since != nil && createdAt.Before(*opts.Since) {
			continue
		}
		if opts.Until != nil && !createdAt.Before(*opts.Until) {
			continue
		}
		if opts.Model != "" && rec.Model != opts.Model {
			continue
		}
		if opts.Provider != "" && rec.Provider != opts.Provider {
			continue
		}
		if opts.Channel != "" && rec.Channel != opts.Channel {
			continue
		}

		dateStr := createdAt.Format("2006-01-02")
		if opts.GroupBy == "monthly" {
			dateStr = createdAt.Format("2006-01")
		}

		key := groupKey{Date: dateStr, Model: rec.Model}
		agg, ok := groups[key]
		if !ok {
			agg = &AggregatedUsage{Date: dateStr, Model: rec.Model}
			groups[key] = agg
		}
		agg.InputTokens += int64(rec.InputTokens)
		agg.OutputTokens += int64(rec.OutputTokens)
		agg.CacheReadTokens += int64(rec.CacheReadTokens)
		agg.CacheWriteTokens += int64(rec.CacheWriteTokens)
		agg.CallCount++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan usage file: %w", err)
	}

	results := make([]AggregatedUsage, 0, len(groups))
	for _, agg := range groups {
		results = append(results, *agg)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Date != results[j].Date {
			return strings.Compare(results[i].Date, results[j].Date) > 0
		}
		return results[i].Model < results[j].Model
	})

	return results, nil
}
