package scheduler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/priyanshujain/openbotkit/store"
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
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)`),
		string(s.Type), s.CronExpr, scheduledAt, s.Task, s.Channel, string(metaJSON), s.Timezone, s.Description,
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
		`SELECT id, type, cron_expr, scheduled_at, task, channel, channel_meta, timezone, description, enabled, last_run_at, last_error, created_at, completed_at
		FROM schedules WHERE enabled = 1 ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list enabled schedules: %w", err)
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func ListDueOneShot(db *store.DB, now time.Time) ([]Schedule, error) {
	rows, err := db.Query(
		db.Rebind(`SELECT id, type, cron_expr, scheduled_at, task, channel, channel_meta, timezone, description, enabled, last_run_at, last_error, created_at, completed_at
		FROM schedules WHERE type = 'one_shot' AND scheduled_at <= ? AND completed_at IS NULL AND enabled = 1 ORDER BY scheduled_at`),
		now.UTC().Format(timeFormat),
	)
	if err != nil {
		return nil, fmt.Errorf("list due one-shot: %w", err)
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func Delete(db *store.DB, id int64) error {
	_, err := db.Exec(db.Rebind("DELETE FROM schedules WHERE id = ?"), id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}

func MarkCompleted(db *store.DB, id int64, completedAt time.Time) error {
	_, err := db.Exec(
		db.Rebind("UPDATE schedules SET completed_at = ?, enabled = 0 WHERE id = ?"),
		completedAt.UTC().Format(timeFormat), id,
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
	var enabled int

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
	s.Enabled = enabled == 1
	s.LastError = lastError.String

	if scheduledAt.Valid {
		t := parseTime(scheduledAt.String)
		s.ScheduledAt = t
	}
	if lastRunAt.Valid {
		s.LastRunAt = parseTime(lastRunAt.String)
	}
	if createdAt.Valid {
		if t := parseTime(createdAt.String); t != nil {
			s.CreatedAt = *t
		}
	}
	if completedAt.Valid {
		s.CompletedAt = parseTime(completedAt.String)
	}

	if metaJSON.Valid && metaJSON.String != "" {
		_ = json.Unmarshal([]byte(metaJSON.String), &s.ChannelMeta)
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

func parseTime(s string) *time.Time {
	for _, f := range []string{
		timeFormat,
		"2006-01-02 15:04:05",
		time.RFC3339,
	} {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}
