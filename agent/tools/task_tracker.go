package tools

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/73ai/openbotkit/service/tasks"
	"github.com/73ai/openbotkit/store"
)

// TaskStatus represents the state of a delegated task.
type TaskStatus string

const (
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

const defaultMaxConcurrent = 3

// TaskRecord holds the state and output of a delegated task.
type TaskRecord struct {
	ID        string     `json:"id"`
	Task      string     `json:"task"`
	Agent     AgentKind  `json:"agent"`
	Status    TaskStatus `json:"status"`
	StartedAt time.Time  `json:"started_at"`
	DoneAt    time.Time  `json:"done_at,omitempty"`
	Output    string     `json:"output,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// TaskTracker manages in-memory state for background delegated tasks.
type TaskTracker struct {
	mu            sync.Mutex
	tasks         map[string]*TaskRecord
	order         []string // insertion order for deterministic listing
	maxConcurrent int
	db            *store.DB // nil for in-memory only
}

// NewTaskTracker creates a tracker with default max concurrency of 3.
func NewTaskTracker() *TaskTracker {
	return &TaskTracker{
		tasks:         make(map[string]*TaskRecord),
		maxConcurrent: defaultMaxConcurrent,
	}
}

// NewPersistentTaskTracker creates a tracker backed by a database.
// It migrates the schema, runs cleanup, and loads existing running tasks.
func NewPersistentTaskTracker(db *store.DB) *TaskTracker {
	if err := tasks.Migrate(db); err != nil {
		slog.Warn("tasks: migrate failed", "error", err)
	}
	tasks.Cleanup(db)

	t := &TaskTracker{
		tasks:         make(map[string]*TaskRecord),
		maxConcurrent: defaultMaxConcurrent,
		db:            db,
	}
	return t
}

// Close closes the underlying database connection if persistent.
func (t *TaskTracker) Close() error {
	if t.db != nil {
		return t.db.Close()
	}
	return nil
}

// Start registers a new running task. Returns error if at max concurrent.
func (t *TaskTracker) Start(id, task string, agent AgentKind) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.runningCountLocked() >= t.maxConcurrent {
		return fmt.Errorf("too many concurrent tasks (max %d)", t.maxConcurrent)
	}
	now := time.Now()
	t.tasks[id] = &TaskRecord{
		ID:        id,
		Task:      task,
		Agent:     agent,
		Status:    TaskRunning,
		StartedAt: now,
	}
	t.order = append(t.order, id)
	if t.db != nil {
		if err := tasks.Insert(t.db, &tasks.TaskRecord{
			ID: id, Task: task, Agent: string(agent),
			Status: "running", StartedAt: now,
		}); err != nil {
			slog.Warn("tasks: db insert failed", "id", id, "error", err)
		}
	}
	return nil
}

// Complete marks a task as completed with output.
func (t *TaskTracker) Complete(id, output string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if rec, ok := t.tasks[id]; ok {
		rec.Status = TaskCompleted
		rec.Output = output
		rec.DoneAt = time.Now()
	}
	if t.db != nil {
		if err := tasks.SetCompleted(t.db, id, output); err != nil {
			slog.Warn("tasks: db set completed failed", "id", id, "error", err)
		}
	}
}

// Fail marks a task as failed with an error message.
func (t *TaskTracker) Fail(id, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if rec, ok := t.tasks[id]; ok {
		rec.Status = TaskFailed
		rec.Error = errMsg
		rec.DoneAt = time.Now()
	}
	if t.db != nil {
		if err := tasks.SetFailed(t.db, id, errMsg); err != nil {
			slog.Warn("tasks: db set failed", "id", id, "error", err)
		}
	}
}

// Get returns a task record by ID. Falls through to DB for cross-session lookup.
func (t *TaskTracker) Get(id string) (*TaskRecord, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.tasks[id]
	if ok {
		copy := *rec
		return &copy, true
	}
	if t.db != nil {
		dbRec, err := tasks.Get(t.db, id)
		if err != nil {
			slog.Warn("tasks: db get failed", "id", id, "error", err)
			return nil, false
		}
		if dbRec != nil {
			return dbTaskToRecord(dbRec), true
		}
	}
	return nil, false
}

// List returns all tasks. When DB is available, returns full cross-session view.
func (t *TaskTracker) List() []*TaskRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.db != nil {
		dbRecs, err := tasks.List(t.db)
		if err != nil {
			slog.Warn("tasks: db list failed", "error", err)
		} else {
			result := make([]*TaskRecord, 0, len(dbRecs))
			for _, r := range dbRecs {
				result = append(result, dbTaskToRecord(r))
			}
			return result
		}
	}
	result := make([]*TaskRecord, 0, len(t.order))
	for _, id := range t.order {
		if rec, ok := t.tasks[id]; ok {
			copy := *rec
			result = append(result, &copy)
		}
	}
	return result
}

func dbTaskToRecord(r *tasks.TaskRecord) *TaskRecord {
	rec := &TaskRecord{
		ID:        r.ID,
		Task:      r.Task,
		Agent:     AgentKind(r.Agent),
		Status:    TaskStatus(r.Status),
		StartedAt: r.StartedAt,
		Output:    r.Output,
		Error:     r.Error,
	}
	if r.DoneAt != nil {
		rec.DoneAt = *r.DoneAt
	}
	return rec
}

// RunningCount returns the number of currently running tasks.
func (t *TaskTracker) RunningCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.runningCountLocked()
}

func (t *TaskTracker) runningCountLocked() int {
	count := 0
	for _, rec := range t.tasks {
		if rec.Status == TaskRunning {
			count++
		}
	}
	return count
}
