package scheduler

import "time"

type ScheduleType string

const (
	Recurring ScheduleType = "recurring"
	OneShot   ScheduleType = "one_shot"
)

type ChannelMeta struct {
	BotToken  string `json:"bot_token,omitempty"`
	OwnerID   int64  `json:"owner_id,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

type Schedule struct {
	ID          int64
	Type        ScheduleType
	CronExpr    string
	ScheduledAt *time.Time
	Task        string
	Channel     string
	ChannelMeta ChannelMeta
	Timezone    string
	Description string
	Enabled     bool
	LastRunAt   *time.Time
	LastError   string
	CreatedAt   time.Time
	CompletedAt *time.Time
}
