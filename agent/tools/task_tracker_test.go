package tools

import (
	"sync"
	"testing"
)

func TestTaskTracker_StartAndGet(t *testing.T) {
	tr := NewTaskTracker()
	if err := tr.Start("t1", "research Go", AgentClaude); err != nil {
		t.Fatalf("Start: %v", err)
	}
	rec, ok := tr.Get("t1")
	if !ok {
		t.Fatal("task not found")
	}
	if rec.ID != "t1" {
		t.Errorf("ID = %q", rec.ID)
	}
	if rec.Task != "research Go" {
		t.Errorf("Task = %q", rec.Task)
	}
	if rec.Agent != AgentClaude {
		t.Errorf("Agent = %q", rec.Agent)
	}
	if rec.Status != TaskRunning {
		t.Errorf("Status = %q", rec.Status)
	}
	if rec.StartedAt.IsZero() {
		t.Error("StartedAt is zero")
	}
	if !rec.DoneAt.IsZero() {
		t.Error("DoneAt should be zero for running task")
	}
}

func TestTaskTracker_Complete(t *testing.T) {
	tr := NewTaskTracker()
	tr.Start("t1", "research", AgentClaude)
	tr.Complete("t1", "done output")
	rec, _ := tr.Get("t1")
	if rec.Status != TaskCompleted {
		t.Errorf("Status = %q", rec.Status)
	}
	if rec.Output != "done output" {
		t.Errorf("Output = %q", rec.Output)
	}
	if rec.DoneAt.IsZero() {
		t.Error("DoneAt should be set")
	}
}

func TestTaskTracker_Fail(t *testing.T) {
	tr := NewTaskTracker()
	tr.Start("t1", "research", AgentClaude)
	tr.Fail("t1", "timeout")
	rec, _ := tr.Get("t1")
	if rec.Status != TaskFailed {
		t.Errorf("Status = %q", rec.Status)
	}
	if rec.Error != "timeout" {
		t.Errorf("Error = %q", rec.Error)
	}
	if rec.DoneAt.IsZero() {
		t.Error("DoneAt should be set")
	}
}

func TestTaskTracker_GetNotFound(t *testing.T) {
	tr := NewTaskTracker()
	_, ok := tr.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestTaskTracker_List(t *testing.T) {
	tr := NewTaskTracker()
	tr.Start("t1", "task1", AgentClaude)
	tr.Start("t2", "task2", AgentGemini)
	tr.Start("t3", "task3", AgentClaude)
	list := tr.List()
	if len(list) != 3 {
		t.Fatalf("got %d tasks, want 3", len(list))
	}
	// Verify insertion order.
	if list[0].ID != "t1" || list[1].ID != "t2" || list[2].ID != "t3" {
		t.Errorf("order: %s, %s, %s", list[0].ID, list[1].ID, list[2].ID)
	}
}

func TestTaskTracker_ListEmpty(t *testing.T) {
	tr := NewTaskTracker()
	list := tr.List()
	if list == nil {
		t.Error("List should return empty slice, not nil")
	}
	if len(list) != 0 {
		t.Errorf("got %d tasks", len(list))
	}
}

func TestTaskTracker_RunningCount(t *testing.T) {
	tr := NewTaskTracker()
	tr.Start("t1", "task1", AgentClaude)
	tr.Start("t2", "task2", AgentClaude)
	tr.Start("t3", "task3", AgentClaude)
	if tr.RunningCount() != 3 {
		t.Errorf("RunningCount = %d, want 3", tr.RunningCount())
	}
	tr.Complete("t1", "done")
	tr.Fail("t2", "err")
	if tr.RunningCount() != 1 {
		t.Errorf("RunningCount = %d, want 1", tr.RunningCount())
	}
}

func TestTaskTracker_MaxConcurrent(t *testing.T) {
	tr := NewTaskTracker()
	tr.Start("t1", "task1", AgentClaude)
	tr.Start("t2", "task2", AgentClaude)
	tr.Start("t3", "task3", AgentClaude)
	err := tr.Start("t4", "task4", AgentClaude)
	if err == nil {
		t.Fatal("expected error at max concurrent")
	}
	tr.Complete("t1", "done")
	if err := tr.Start("t4", "task4", AgentClaude); err != nil {
		t.Fatalf("Start after completion: %v", err)
	}
}

func TestTaskTracker_ConcurrentAccess(t *testing.T) {
	tr := NewTaskTracker()
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := string(rune('a' + n))
			tr.Start(id, "task", AgentClaude)
			tr.List()
			tr.RunningCount()
			tr.Complete(id, "done")
			tr.Get(id)
		}(i)
	}
	wg.Wait()
}

func TestTaskTracker_DuplicateID(t *testing.T) {
	tr := NewTaskTracker()
	tr.Start("dup", "first task", AgentClaude)
	tr.Start("dup", "second task", AgentClaude)
	rec, _ := tr.Get("dup")
	if rec.Task != "second task" {
		t.Errorf("Task = %q, expected second task to overwrite", rec.Task)
	}
}

func TestTaskTracker_GetReturnsCopy(t *testing.T) {
	tr := NewTaskTracker()
	tr.Start("t1", "task", AgentClaude)
	rec, _ := tr.Get("t1")
	rec.Status = TaskCompleted // mutate the copy
	original, _ := tr.Get("t1")
	if original.Status != TaskRunning {
		t.Error("modifying Get result should not affect tracker state")
	}
}

func TestTaskTracker_CompleteNonexistent(t *testing.T) {
	tr := NewTaskTracker()
	tr.Complete("ghost", "output") // should not panic
}

func TestTaskTracker_FailNonexistent(t *testing.T) {
	tr := NewTaskTracker()
	tr.Fail("ghost", "error") // should not panic
}
