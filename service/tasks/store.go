package tasks

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/73ai/openbotkit/store"
)

const timeFormat = "2006-01-02T15:04:05Z"

type TaskRecord struct {
	ID        string
	Task      string
	Agent     string
	Status    string
	StartedAt time.Time
	DoneAt    *time.Time
	Output    string
	Error     string
}

func Insert(db *store.DB, r *TaskRecord) error {
	_, err := db.Exec(
		db.Rebind(`INSERT INTO tasks (id, task, agent, status, started_at) VALUES (?, ?, ?, ?, ?)`),
		r.ID, r.Task, r.Agent, r.Status, r.StartedAt.UTC().Format(timeFormat),
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func SetCompleted(db *store.DB, id, output string) error {
	_, err := db.Exec(
		db.Rebind(`UPDATE tasks SET status = 'completed', output = ?, done_at = ? WHERE id = ?`),
		output, time.Now().UTC().Format(timeFormat), id,
	)
	if err != nil {
		return fmt.Errorf("set task completed: %w", err)
	}
	return nil
}

func SetFailed(db *store.DB, id, errMsg string) error {
	_, err := db.Exec(
		db.Rebind(`UPDATE tasks SET status = 'failed', error = ?, done_at = ? WHERE id = ?`),
		errMsg, time.Now().UTC().Format(timeFormat), id,
	)
	if err != nil {
		return fmt.Errorf("set task failed: %w", err)
	}
	return nil
}

func Get(db *store.DB, id string) (*TaskRecord, error) {
	row := db.QueryRow(
		db.Rebind(`SELECT id, task, agent, status, started_at, done_at, output, error FROM tasks WHERE id = ?`),
		id,
	)
	return scanTask(row)
}

func List(db *store.DB) ([]*TaskRecord, error) {
	rows, err := db.Query(`SELECT id, task, agent, status, started_at, done_at, output, error FROM tasks ORDER BY started_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	return scanTasks(rows)
}

func CountRunning(db *store.DB) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status = 'running'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count running: %w", err)
	}
	return count, nil
}

func DeleteOlderThan(db *store.DB, before time.Time) (int64, error) {
	res, err := db.Exec(
		db.Rebind(`DELETE FROM tasks WHERE status IN ('completed', 'failed') AND done_at < ?`),
		before.UTC().Format(timeFormat),
	)
	if err != nil {
		return 0, fmt.Errorf("delete old tasks: %w", err)
	}
	return res.RowsAffected()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTask(row scannable) (*TaskRecord, error) {
	var r TaskRecord
	var startedAt, doneAt, output, errMsg sql.NullString

	err := row.Scan(&r.ID, &r.Task, &r.Agent, &r.Status, &startedAt, &doneAt, &output, &errMsg)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}

	r.Output = output.String
	r.Error = errMsg.String

	if startedAt.Valid {
		t, err := parseTime(startedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse started_at: %w", err)
		}
		r.StartedAt = *t
	}
	if doneAt.Valid {
		t, err := parseTime(doneAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse done_at: %w", err)
		}
		r.DoneAt = t
	}

	return &r, nil
}

func scanTasks(rows *sql.Rows) ([]*TaskRecord, error) {
	var result []*TaskRecord
	for rows.Next() {
		r, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
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
