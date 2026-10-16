package telegram

import (
	"testing"

	"github.com/73ai/openbotkit/store"
	"github.com/73ai/openbotkit/store/migratetest"
)

const preDialogChats = `
INSERT INTO telegram_chats (chat_id, type, title, username, access_hash, is_group, is_channel, last_message_at) VALUES
	(-1001498765432, 'supergroup', 'Bangalore Gophers', 'blrgophers', 8410236659178234511, TRUE, FALSE, '2026-09-28 17:42:10'),
	(777000, 'user', 'Telegram', '', 0, FALSE, FALSE, '2026-09-01 08:00:00'),
	(5012345678, 'user', 'Anika Rao', 'anikarao', -3127745501928374655, FALSE, FALSE, '2026-10-02 22:15:31');
`

func TestMigrations(t *testing.T) {
	migratetest.Run(t, migratetest.Suite{
		Source:  "telegram",
		Migrate: Migrate,
		Steps:   migrations,
		Fixtures: []migratetest.Fixture{{
			Name: "pre_dialog",
			SQLite: `CREATE TABLE telegram_chats (
	chat_id INTEGER PRIMARY KEY,
	type TEXT,
	title TEXT,
	username TEXT,
	access_hash INTEGER DEFAULT 0,
	is_group INTEGER DEFAULT 0,
	is_channel INTEGER DEFAULT 0,
	last_message_at DATETIME
);` + preDialogChats,
			Postgres: `CREATE TABLE telegram_chats (
	chat_id BIGINT PRIMARY KEY,
	type TEXT,
	title TEXT,
	username TEXT,
	access_hash BIGINT DEFAULT 0,
	is_group BOOLEAN DEFAULT FALSE,
	is_channel BOOLEAN DEFAULT FALSE,
	last_message_at TIMESTAMPTZ
);` + preDialogChats,
			Preserve: []string{"SELECT chat_id, type, title, username, access_hash, is_group, is_channel, last_message_at FROM telegram_chats ORDER BY chat_id"},
		}},
		Check: func(t *testing.T, db *store.DB, workers int) {
			if got := migratetest.Dump(t, db, "SELECT chat_id FROM telegram_chats WHERE NOT is_current_dialog"); len(got) > 0 {
				t.Fatalf("chats from before dialog tracking not marked current: %v", got)
			}
		},
	})
}
