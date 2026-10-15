package scheduler

import (
	"testing"

	"github.com/73ai/openbotkit/store"
	"github.com/73ai/openbotkit/store/migratetest"
)

const baselineSchedules = `
INSERT INTO schedules (id, type, cron_expr, task, channel, channel_meta, timezone, description, enabled, last_run_at, created_at) VALUES
	(12, 'recurring', '30 8 * * 1-5', 'Summarise unread work email and WhatsApp mentions', 'telegram', '{"chat_id":5012345678}', 'Asia/Kolkata', 'weekday briefing', TRUE, '2026-10-14 03:00:04', '2026-08-19 18:22:41');
INSERT INTO schedules (id, type, scheduled_at, task, channel, timezone, description, enabled, created_at, completed_at) VALUES
	(15, 'one_shot', '2026-10-03 11:30:00', 'Remind me to renew the car insurance', 'telegram', 'Asia/Kolkata', 'insurance reminder', FALSE, '2026-09-30 21:05:12', '2026-10-03 11:30:02');
`

const budgetColumns = `
ALTER TABLE schedules ADD COLUMN model_tier TEXT DEFAULT 'fast';
ALTER TABLE schedules ADD COLUMN max_budget_usd REAL DEFAULT 0;
UPDATE schedules SET model_tier = 'smart', max_budget_usd = 0.5 WHERE id = 12;
`

var scheduleRows = []string{"SELECT id, type, cron_expr, scheduled_at, task, channel, channel_meta, timezone, description, enabled, last_run_at, last_error, created_at, completed_at FROM schedules ORDER BY id"}

func TestMigrations(t *testing.T) {
	migratetest.Run(t, migratetest.Suite{
		Source:  "scheduler",
		Migrate: Migrate,
		Steps:   migrations,
		Fixtures: []migratetest.Fixture{
			{Name: "baseline", SQLite: schemaSQLite + baselineSchedules, Postgres: schemaPostgres + baselineSchedules, Preserve: scheduleRows},
			{
				Name:     "budget_columns",
				SQLite:   schemaSQLite + baselineSchedules + budgetColumns,
				Postgres: schemaPostgres + baselineSchedules + budgetColumns,
				Preserve: append(scheduleRows, "SELECT id, model_tier, max_budget_usd FROM schedules ORDER BY id"),
			},
		},
		Check: func(t *testing.T, db *store.DB, workers int) {
			if got := migratetest.Dump(t, db, "SELECT id FROM schedules WHERE model_tier IS NULL OR max_budget_usd IS NULL OR last_trigger_id IS NULL"); len(got) > 0 {
				t.Fatalf("schedules missing column defaults after migration: %v", got)
			}
		},
	})
}
