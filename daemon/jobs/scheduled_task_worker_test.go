package jobs

import (
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/73ai/openbotkit/config"
)

func TestScheduledTaskWorkerNextRetry(t *testing.T) {
	tests := []struct {
		name   string
		errors []rivertype.AttemptError
		minDur time.Duration
		maxDur time.Duration
	}{
		{
			name:   "no errors defaults to 15min",
			errors: nil,
			minDur: 14 * time.Minute,
			maxDur: 16 * time.Minute,
		},
		// Auth and context-window errors are cancelled in Work() via
		// river.JobCancel, so NextRetry is never called for those.
		{
			name:   "rate limit 429 delays 30min",
			errors: []rivertype.AttemptError{{Error: "API error (HTTP 429): rate limited"}},
			minDur: 29 * time.Minute,
			maxDur: 31 * time.Minute,
		},
		{
			name:   "server error 500 delays 10min",
			errors: []rivertype.AttemptError{{Error: "API error (HTTP 500): internal server error"}},
			minDur: 9 * time.Minute,
			maxDur: 11 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &ScheduledTaskWorker{}
			job := &river.Job[ScheduledTaskArgs]{
				JobRow: &rivertype.JobRow{Errors: tt.errors},
			}
			now := time.Now()
			retryAt := w.NextRetry(job)
			minExpected := now.Add(tt.minDur)
			maxExpected := now.Add(tt.maxDur)
			if retryAt.Before(minExpected) || retryAt.After(maxExpected) {
				t.Errorf("NextRetry = %v, want between %v and %v", retryAt, minExpected, maxExpected)
			}
		})
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
