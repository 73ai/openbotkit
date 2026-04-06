package hooks

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/73ai/openbotkit/store"
)

const timeFormat = "2006-01-02T15:04:05Z"

func Create(db *store.DB, h *EventHook) (int64, error) {
	res, err := db.Exec(
		db.Rebind(`INSERT INTO event_hooks (event_type, prompt, channel, model_tier, enabled)
			VALUES (?, ?, ?, ?, ?)`),
		h.EventType, h.Prompt, h.Channel, h.ModelTier, 1,
	)
	if err != nil {
		return 0, fmt.Errorf("insert event_hook: %w", err)
	}
	return res.LastInsertId()
}

func ListEnabled(db *store.DB, eventType string) ([]EventHook, error) {
	rows, err := db.Query(
		db.Rebind(`SELECT id, event_type, prompt, channel, model_tier, enabled,
			last_run_at, last_error, created_at
			FROM event_hooks WHERE event_type = ? AND enabled = ?
			ORDER BY created_at`),
		eventType, 1,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled hooks: %w", err)
	}
	defer rows.Close()
	return scanHooks(rows)
}

func UpdateLastRun(db *store.DB, id int64, runAt time.Time, lastErr string) error {
	var errVal sql.NullString
	if lastErr != "" {
		errVal = sql.NullString{String: lastErr, Valid: true}
	}
	_, err := db.Exec(
		db.Rebind("UPDATE event_hooks SET last_run_at = ?, last_error = ? WHERE id = ?"),
		runAt.UTC().Format(timeFormat), errVal, id,
	)
	if err != nil {
		return fmt.Errorf("update last run: %w", err)
	}
	return nil
}

func scanHooks(rows *sql.Rows) ([]EventHook, error) {
	var result []EventHook
	for rows.Next() {
		h, err := scanHook(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *h)
	}
	return result, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanHook(row scannable) (*EventHook, error) {
	var h EventHook
	var lastRunAt, lastError, createdAt sql.NullString
	var enabled any

	err := row.Scan(
		&h.ID, &h.EventType, &h.Prompt, &h.Channel, &h.ModelTier,
		&enabled, &lastRunAt, &lastError, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("hook not found")
		}
		return nil, fmt.Errorf("scan hook: %w", err)
	}

	switch v := enabled.(type) {
	case int64:
		h.Enabled = v == 1
	case bool:
		h.Enabled = v
	}
	h.LastError = lastError.String
	if lastRunAt.Valid {
		t, _ := time.Parse(timeFormat, lastRunAt.String)
		h.LastRunAt = &t
	}
	if createdAt.Valid {
		t, _ := time.Parse(timeFormat, createdAt.String)
		h.CreatedAt = t
	}
	return &h, nil
}

func Get(db *store.DB, id int64) (*EventHook, error) {
	row := db.QueryRow(
		db.Rebind(`SELECT id, event_type, prompt, channel, model_tier, enabled,
			last_run_at, last_error, created_at
			FROM event_hooks WHERE id = ?`),
		id,
	)
	return scanHook(row)
}
