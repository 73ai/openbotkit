package jobs

import (
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/73ai/openbotkit/config"
)

func TestScheduledTaskWorkerNextRetryAt(t *testing.T) {
	w := &ScheduledTaskWorker{}
	before := time.Now().Add(14 * time.Minute)
	retryAt := w.NextRetryAt(&river.Job[ScheduledTaskArgs]{})
	after := time.Now().Add(16 * time.Minute)

	if retryAt.Before(before) || retryAt.After(after) {
		t.Errorf("NextRetryAt should be ~15 min from now, got %v", retryAt)
	}
}

func TestScheduledTaskWorkerRunAgentNilCfg(t *testing.T) {
	w := &ScheduledTaskWorker{}
	_, err := w.runAgent(t.Context(), "test task")
	if err == nil {
		t.Fatal("expected error when cfg is nil")
	}
}

func TestScheduledTaskWorkerRunAgentNoModel(t *testing.T) {
	w := &ScheduledTaskWorker{Cfg: config.Default()}
	_, err := w.runAgent(t.Context(), "test task")
	if err == nil {
		t.Fatal("expected error when no model configured")
	}
}
