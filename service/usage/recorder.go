package usage

import (
	"log/slog"

	"github.com/73ai/openbotkit/provider"
)

// Recorder implements agent.UsageRecorder by appending to a JSONL file.
type Recorder struct {
	path      string
	provider  string
	channel   string
	sessionID string
}

// NewRecorder creates a Recorder that appends usage to the given JSONL file.
func NewRecorder(path, providerName, channel, sessionID string) *Recorder {
	return &Recorder{
		path:      path,
		provider:  providerName,
		channel:   channel,
		sessionID: sessionID,
	}
}

// Close is a no-op since each Record call opens/closes the file.
func (r *Recorder) Close() {}

func (r *Recorder) RecordUsage(model string, usage provider.Usage) {
	err := Record(r.path, UsageRecord{
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
