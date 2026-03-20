package tasks

import (
	"log/slog"
	"time"

	"github.com/73ai/openbotkit/store"
)

const retentionDays = 7

func Cleanup(db *store.DB) {
	cutoff := time.Now().UTC().Add(-retentionDays * 24 * time.Hour)
	n, err := DeleteOlderThan(db, cutoff)
	if err != nil {
		slog.Warn("tasks cleanup failed", "error", err)
		return
	}
	if n > 0 {
		slog.Info("tasks cleanup", "deleted", n)
	}
}
