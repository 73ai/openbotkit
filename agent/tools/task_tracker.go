package tools

import (
	"fmt"
	"sync"
	"time"
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
}

// NewTaskTracker creates a tracker with default max concurrency of 3.
func NewTaskTracker() *TaskTracker {
	return &TaskTracker{
		tasks:         make(map[string]*TaskRecord),
		maxConcurrent: defaultMaxConcurrent,
	}
}

// Start registers a new running task. Returns error if at max concurrent.
func (t *TaskTracker) Start(id, task string, agent AgentKind) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.runningCountLocked() >= t.maxConcurrent {
		return fmt.Errorf("too many concurrent tasks (max %d)", t.maxConcurrent)
	}
	t.tasks[id] = &TaskRecord{
		ID:        id,
		Task:      task,
		Agent:     agent,
		Status:    TaskRunning,
		StartedAt: time.Now(),
	}
	t.order = append(t.order, id)
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
}

// Get returns a task record by ID.
func (t *TaskTracker) Get(id string) (*TaskRecord, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.tasks[id]
	if !ok {
		return nil, false
	}
	copy := *rec
	return &copy, true
}

// List returns all tasks in insertion order.
func (t *TaskTracker) List() []*TaskRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]*TaskRecord, 0, len(t.order))
	for _, id := range t.order {
		if rec, ok := t.tasks[id]; ok {
			copy := *rec
			result = append(result, &copy)
		}
	}
	return result
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
