package usage

import (
	"log/slog"

	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/store"
)

// Recorder implements agent.UsageRecorder by writing to the usage store.
type Recorder struct {
	db        *store.DB
	provider  string
	channel   string
	sessionID string
}

// NewRecorder creates a Recorder that writes usage to the given database.
func NewRecorder(db *store.DB, providerName, channel, sessionID string) *Recorder {
	return &Recorder{
		db:        db,
		provider:  providerName,
		channel:   channel,
		sessionID: sessionID,
	}
}

// Close releases the underlying database connection.
func (r *Recorder) Close() {
	r.db.Close()
}

func (r *Recorder) RecordUsage(model string, usage provider.Usage) {
	err := Record(r.db, UsageRecord{
		Provider:         r.provider,
		Model:            model,
		Channel:          r.channel,
		SessionID:        r.sessionID,
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
	})
	if err != nil {
		slog.Debug("usage: failed to record", "error", err)
	}
}
