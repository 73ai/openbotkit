package usage

import (
	"fmt"
	"time"

	"github.com/73ai/openbotkit/store"
)

const schemaSQLite = `
CREATE TABLE IF NOT EXISTS usage_records (
	id                 INTEGER PRIMARY KEY AUTOINCREMENT,
	provider           TEXT NOT NULL,
	model              TEXT NOT NULL,
	channel            TEXT NOT NULL DEFAULT '',
	session_id         TEXT NOT NULL DEFAULT '',
	input_tokens       INTEGER NOT NULL DEFAULT 0,
	output_tokens      INTEGER NOT NULL DEFAULT 0,
	cache_read_tokens  INTEGER NOT NULL DEFAULT 0,
	cache_write_tokens INTEGER NOT NULL DEFAULT 0,
	created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_usage_created_at ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_model ON usage_records(model);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS usage_records (
	id                 BIGSERIAL PRIMARY KEY,
	provider           TEXT NOT NULL,
	model              TEXT NOT NULL,
	channel            TEXT NOT NULL DEFAULT '',
	session_id         TEXT NOT NULL DEFAULT '',
	input_tokens       INTEGER NOT NULL DEFAULT 0,
	output_tokens      INTEGER NOT NULL DEFAULT 0,
	cache_read_tokens  INTEGER NOT NULL DEFAULT 0,
	cache_write_tokens INTEGER NOT NULL DEFAULT 0,
	created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_created_at ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_model ON usage_records(model);
`

func Migrate(db *store.DB) error {
	schema := schemaSQLite
	if db.IsPostgres() {
		schema = schemaPostgres
	}
	_, err := db.Exec(schema)
	return err
}

// UsageRecord represents a single LLM API call's token usage.
type UsageRecord struct {
	Provider         string
	Model            string
	Channel          string
	SessionID        string
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
}

func Record(db *store.DB, rec UsageRecord) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO usage_records
			(provider, model, channel, session_id, input_tokens, output_tokens, cache_read_tokens, cache_write_tokens)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		rec.Provider, rec.Model, rec.Channel, rec.SessionID,
		rec.InputTokens, rec.OutputTokens, rec.CacheReadTokens, rec.CacheWriteTokens,
	)
	if err != nil {
		return fmt.Errorf("insert usage record: %w", err)
	}
	return nil
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

func Query(db *store.DB, opts QueryOpts) ([]AggregatedUsage, error) {
	dateExpr := "DATE(created_at)"
	if opts.GroupBy == "monthly" {
		if db.IsSQLite() {
			dateExpr = "STRFTIME('%Y-%m', created_at)"
		} else {
			dateExpr = "TO_CHAR(created_at, 'YYYY-MM')"
		}
	}

	query := fmt.Sprintf(`SELECT %s AS date_group, model,
		SUM(input_tokens), SUM(output_tokens),
		SUM(cache_read_tokens), SUM(cache_write_tokens),
		COUNT(*)
		FROM usage_records WHERE 1=1`, dateExpr)
	var args []any

	if opts.Since != nil {
		query += " AND created_at >= ?"
		args = append(args, opts.Since.Format("2006-01-02"))
	}
	if opts.Until != nil {
		query += " AND created_at < ?"
		args = append(args, opts.Until.Format("2006-01-02"))
	}
	if opts.Model != "" {
		query += " AND model = ?"
		args = append(args, opts.Model)
	}
	if opts.Provider != "" {
		query += " AND provider = ?"
		args = append(args, opts.Provider)
	}
	if opts.Channel != "" {
		query += " AND channel = ?"
		args = append(args, opts.Channel)
	}

	query += fmt.Sprintf(" GROUP BY %s, model ORDER BY date_group DESC, model", dateExpr)
	query = db.Rebind(query)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query usage: %w", err)
	}
	defer rows.Close()

	var results []AggregatedUsage
	for rows.Next() {
		var r AggregatedUsage
		if err := rows.Scan(&r.Date, &r.Model, &r.InputTokens, &r.OutputTokens,
			&r.CacheReadTokens, &r.CacheWriteTokens, &r.CallCount); err != nil {
			return nil, fmt.Errorf("scan usage row: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
