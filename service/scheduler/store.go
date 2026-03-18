package scheduler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/73ai/openbotkit/store"
)

const timeFormat = "2006-01-02T15:04:05Z"

func Create(db *store.DB, s *Schedule) (int64, error) {
	metaJSON, err := json.Marshal(s.ChannelMeta)
	if err != nil {
		return 0, fmt.Errorf("marshal channel_meta: %w", err)
	}

	var scheduledAt sql.NullString
	if s.ScheduledAt != nil {
		scheduledAt = sql.NullString{String: s.ScheduledAt.UTC().Format(timeFormat), Valid: true}
	}

	res, err := db.Exec(
		db.Rebind(`INSERT INTO schedules (type, cron_expr, scheduled_at, task, channel, channel_meta, timezone, description, enabled)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		string(s.Type), s.CronExpr, scheduledAt, s.Task, s.Channel, string(metaJSON), s.Timezone, s.Description, 1,
	)
	if err != nil {
		return 0, fmt.Errorf("insert schedule: %w", err)
	}
	return res.LastInsertId()
}

func Get(db *store.DB, id int64) (*Schedule, error) {
	row := db.QueryRow(
		db.Rebind(`SELECT id, type, cron_expr, scheduled_at, task, channel, channel_meta, timezone, description, enabled, last_run_at, last_error, created_at, completed_at
			FROM schedules WHERE id = ?`),
		id,
	)
	return scanSchedule(row)
}

func List(db *store.DB) ([]Schedule, error) {
	rows, err := db.Query(
		`SELECT id, type, cron_expr, scheduled_at, task, channel, channel_meta, timezone, description, enabled, last_run_at, last_error, created_at, completed_at
		FROM schedules ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func ListEnabled(db *store.DB) ([]Schedule, error) {
	rows, err := db.Query(
		db.Rebind(`SELECT id, type, cron_expr, scheduled_at, task, channel, channel_meta, timezone, description, enabled, last_run_at, last_error, created_at, completed_at
		FROM schedules WHERE enabled = ? ORDER BY created_at`), 1)
	if err != nil {
		return nil, fmt.Errorf("list enabled schedules: %w", err)
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func ListDueOneShot(db *store.DB, now time.Time) ([]Schedule, error) {
	rows, err := db.Query(
		db.Rebind(`SELECT id, type, cron_expr, scheduled_at, task, channel, channel_meta, timezone, description, enabled, last_run_at, last_error, created_at, completed_at
		FROM schedules WHERE type = 'one_shot' AND scheduled_at <= ? AND completed_at IS NULL AND enabled = ? ORDER BY scheduled_at`),
		now.UTC().Format(timeFormat), 1,
	)
	if err != nil {
		return nil, fmt.Errorf("list due one-shot: %w", err)
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func Disable(db *store.DB, id int64) error {
	_, err := db.Exec(db.Rebind("UPDATE schedules SET enabled = ? WHERE id = ?"), 0, id)
	if err != nil {
		return fmt.Errorf("disable schedule: %w", err)
	}
	return nil
}

func Delete(db *store.DB, id int64) error {
	res, err := db.Exec(db.Rebind("DELETE FROM schedules WHERE id = ?"), id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("schedule %d not found", id)
	}
	return nil
}

func MarkCompleted(db *store.DB, id int64, completedAt time.Time) error {
	_, err := db.Exec(
		db.Rebind("UPDATE schedules SET completed_at = ?, enabled = ? WHERE id = ?"),
		completedAt.UTC().Format(timeFormat), 0, id,
	)
	if err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}
	return nil
}

func UpdateLastRun(db *store.DB, id int64, runAt time.Time, lastErr string) error {
	var errVal sql.NullString
	if lastErr != "" {
		errVal = sql.NullString{String: lastErr, Valid: true}
	}
	_, err := db.Exec(
		db.Rebind("UPDATE schedules SET last_run_at = ?, last_error = ? WHERE id = ?"),
		runAt.UTC().Format(timeFormat), errVal, id,
	)
	if err != nil {
		return fmt.Errorf("update last run: %w", err)
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSchedule(row scannable) (*Schedule, error) {
	var s Schedule
	var typ, cronExpr, scheduledAt, channel, metaJSON, tz sql.NullString
	var desc, lastRunAt, lastError, createdAt, completedAt sql.NullString
	var enabled any

	err := row.Scan(
		&s.ID, &typ, &cronExpr, &scheduledAt, &s.Task, &channel, &metaJSON,
		&tz, &desc, &enabled, &lastRunAt, &lastError, &createdAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("schedule not found")
		}
		return nil, fmt.Errorf("scan schedule: %w", err)
	}

	s.Type = ScheduleType(typ.String)
	s.CronExpr = cronExpr.String
	s.Channel = channel.String
	s.Timezone = tz.String
	s.Description = desc.String
	// Handle both SQLite (integer) and Postgres (boolean) column types.
	switch v := enabled.(type) {
	case int64:
		s.Enabled = v == 1
	case bool:
		s.Enabled = v
	}
	s.LastError = lastError.String

	if scheduledAt.Valid {
		t, err := parseTime(scheduledAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse scheduled_at: %w", err)
		}
		s.ScheduledAt = t
	}
	if lastRunAt.Valid {
		t, err := parseTime(lastRunAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse last_run_at: %w", err)
		}
		s.LastRunAt = t
	}
	if createdAt.Valid {
		t, err := parseTime(createdAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		s.CreatedAt = *t
	}
	if completedAt.Valid {
		t, err := parseTime(completedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse completed_at: %w", err)
		}
		s.CompletedAt = t
	}

	if metaJSON.Valid && metaJSON.String != "" {
		if err := json.Unmarshal([]byte(metaJSON.String), &s.ChannelMeta); err != nil {
			return nil, fmt.Errorf("unmarshal channel_meta: %w", err)
		}
	}

	return &s, nil
}

func scanSchedules(rows *sql.Rows) ([]Schedule, error) {
	var result []Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *s)
	}
	return result, rows.Err()
}

func parseTime(s string) (*time.Time, error) {
	for _, f := range []string{
		timeFormat,
		"2006-01-02 15:04:05",
		time.RFC3339,
	} {
		if t, err := time.Parse(f, s); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("unrecognised time format: %q", s)
}
