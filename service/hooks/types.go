package hooks

import "time"

type EventHook struct {
	ID        int64
	EventType string // "gmail_sync", "backup_complete"
	Prompt    string // combined classify+notify prompt
	Channel   string // "telegram" (resolved via ChannelRegistry)
	ModelTier string // "nano" or "fast"
	Enabled   bool
	LastRunAt *time.Time
	LastError string
	CreatedAt time.Time
}
