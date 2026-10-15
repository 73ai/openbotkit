package whatsapp

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/73ai/openbotkit/store"
)

const legacySchemaSQLite = `
CREATE TABLE IF NOT EXISTS whatsapp_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	message_id TEXT NOT NULL,
	chat_jid TEXT NOT NULL,
	sender_jid TEXT,
	sender_name TEXT,
	text TEXT,
	timestamp DATETIME,
	media_type TEXT,
	media_path TEXT,
	is_group INTEGER DEFAULT 0,
	is_from_me INTEGER DEFAULT 0,
	reply_to_id TEXT,
	synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(message_id, chat_jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_chats (
	jid TEXT PRIMARY KEY,
	name TEXT,
	is_group INTEGER DEFAULT 0,
	last_message_at DATETIME
);

CREATE TABLE IF NOT EXISTS whatsapp_contacts (
	jid TEXT PRIMARY KEY,
	phone TEXT,
	first_name TEXT,
	full_name TEXT,
	push_name TEXT,
	business_name TEXT,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_sender ON whatsapp_messages(sender_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
`

const legacySchemaPostgres = `
CREATE TABLE IF NOT EXISTS whatsapp_messages (
	id BIGSERIAL PRIMARY KEY,
	message_id TEXT NOT NULL,
	chat_jid TEXT NOT NULL,
	sender_jid TEXT,
	sender_name TEXT,
	text TEXT,
	timestamp TIMESTAMPTZ,
	media_type TEXT,
	media_path TEXT,
	is_group BOOLEAN DEFAULT FALSE,
	is_from_me BOOLEAN DEFAULT FALSE,
	reply_to_id TEXT,
	synced_at TIMESTAMPTZ DEFAULT NOW(),
	UNIQUE(message_id, chat_jid)
);

CREATE TABLE IF NOT EXISTS whatsapp_chats (
	jid TEXT PRIMARY KEY,
	name TEXT,
	is_group BOOLEAN DEFAULT FALSE,
	last_message_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS whatsapp_contacts (
	jid TEXT PRIMARY KEY,
	phone TEXT,
	first_name TEXT,
	full_name TEXT,
	push_name TEXT,
	business_name TEXT,
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_chat ON whatsapp_messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_sender ON whatsapp_messages(sender_jid);
CREATE INDEX IF NOT EXISTS idx_whatsapp_contacts_phone ON whatsapp_contacts(phone);
`

const legacyData = `
INSERT INTO whatsapp_messages (message_id, chat_jid, sender_jid, sender_name, text, timestamp, is_group, is_from_me) VALUES
	('3EB0A1', '120363041234567890@g.us', '919812345678@s.whatsapp.net', 'Ravi', 'standup moved to 11', '2026-03-02 09:14:00', TRUE, FALSE),
	('3EB0A2', '120363041234567890@g.us', '919876543210@s.whatsapp.net', '', 'ok', '2026-03-02 09:15:00', TRUE, TRUE),
	('BAE5F3', '919812345678@s.whatsapp.net', '919812345678@s.whatsapp.net', 'Ravi', 'call me when free', '2026-03-03 18:40:00', FALSE, FALSE);
INSERT INTO whatsapp_messages (message_id, chat_jid, sender_jid, text, timestamp, media_type, media_path, reply_to_id) VALUES
	('BAE5F4', '919812345678@s.whatsapp.net', '919876543210@s.whatsapp.net', '', '2026-03-03 18:42:00', 'image', '/tmp/media/BAE5F4.jpg', 'BAE5F3');
INSERT INTO whatsapp_chats (jid, name, is_group, last_message_at) VALUES
	('120363041234567890@g.us', 'Platform Team', TRUE, '2026-03-02 09:15:00'),
	('919812345678@s.whatsapp.net', 'Ravi', FALSE, '2026-03-03 18:42:00');
INSERT INTO whatsapp_contacts (jid, phone, first_name, full_name, push_name, business_name) VALUES
	('919812345678@s.whatsapp.net', '919812345678', 'Ravi', 'Ravi Kumar', 'ravi', ''),
	('14155550100@s.whatsapp.net', '14155550100', '', '', 'Acme Support', 'Acme Inc');
`

type dbFactory func(t *testing.T) *store.DB

func legacySQLite(t *testing.T) dbFactory {
	dsn := filepath.Join(t.TempDir(), "data.db")
	open := func(t *testing.T) *store.DB {
		db, err := store.Open(store.SQLiteConfig(dsn))
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		t.Cleanup(func() { db.Close() })
		return db
	}
	if _, err := open(t).Exec(legacySchemaSQLite + legacyData); err != nil {
		t.Fatalf("seed legacy sqlite: %v", err)
	}
	return open
}

func legacyPostgres(t *testing.T) dbFactory {
	if testing.Short() {
		t.Skip("skipping postgres test in short mode")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("skipping: Docker is not running")
	}
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			Env:          map[string]string{"POSTGRES_PASSWORD": "obk", "POSTGRES_DB": "obk"},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { container.Terminate(ctx) })
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("postgres host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("postgres port: %v", err)
	}
	dsn := fmt.Sprintf("postgres://postgres:obk@%s:%s/obk?sslmode=disable", host, port.Port())
	open := func(t *testing.T) *store.DB {
		db, err := store.Open(store.PostgresConfig(dsn))
		if err != nil {
			t.Fatalf("open postgres: %v", err)
		}
		t.Cleanup(func() { db.Close() })
		return db
	}
	if _, err := open(t).Exec(legacySchemaPostgres + legacyData); err != nil {
		t.Fatalf("seed legacy postgres: %v", err)
	}
	return open
}

func dump(t *testing.T, db *store.DB, query string) []string {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("dump %q: %v", query, err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan %q: %v", query, err)
		}
		for i, v := range vals {
			if b, ok := v.([]byte); ok {
				vals[i] = string(b)
			}
		}
		out = append(out, strings.TrimSuffix(fmt.Sprintln(vals...), "\n"))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows %q: %v", query, err)
	}
	return out
}

var legacyTableQueries = []string{
	"SELECT id, message_id, chat_jid, sender_jid, sender_name, text, timestamp, media_type, media_path, is_group, is_from_me, reply_to_id, synced_at FROM whatsapp_messages ORDER BY id",
	"SELECT jid, name, is_group, last_message_at FROM whatsapp_chats ORDER BY jid",
	"SELECT jid, phone, first_name, full_name, push_name, business_name, updated_at FROM whatsapp_contacts ORDER BY jid",
}

func indexNames(t *testing.T, db *store.DB) []string {
	t.Helper()
	query := "SELECT name FROM sqlite_master WHERE type = 'index' AND name LIKE 'idx_whatsapp_%' ORDER BY name"
	if db.IsPostgres() {
		query = "SELECT indexname FROM pg_indexes WHERE indexname LIKE 'idx_whatsapp_%' ORDER BY indexname"
	}
	return dump(t, db, query)
}

func testMigrateLegacy(t *testing.T, open dbFactory) {
	db := open(t)
	var before [][]string
	for _, q := range legacyTableQueries {
		before = append(before, dump(t, db, q))
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate legacy: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	for i, q := range legacyTableQueries {
		if after := dump(t, db, q); !reflect.DeepEqual(after, before[i]) {
			t.Fatalf("rows changed by migration\nbefore: %v\nafter:  %v", before[i], after)
		}
	}
	for _, table := range []string{"whatsapp_messages", "whatsapp_chats", "whatsapp_contacts"} {
		if got := dump(t, db, "SELECT DISTINCT account FROM "+table); !reflect.DeepEqual(got, []string{DefaultAccount}) {
			t.Fatalf("%s accounts after migration = %v", table, got)
		}
	}
	wantIndexes := []string{
		"idx_whatsapp_contacts_phone",
		"idx_whatsapp_messages_account",
		"idx_whatsapp_messages_chat",
		"idx_whatsapp_messages_sender",
		"idx_whatsapp_messages_timestamp",
	}
	if got := indexNames(t, db); !reflect.DeepEqual(got, wantIndexes) {
		t.Fatalf("indexes = %v, want %v", got, wantIndexes)
	}
	if db.IsSQLite() {
		if got := dump(t, db, "SELECT name FROM sqlite_master WHERE name LIKE '%_legacy'"); len(got) != 0 {
			t.Fatalf("legacy tables left behind: %v", got)
		}
	}

	work := &Message{Account: "work", MessageID: "3EB0A1", ChatJID: "120363041234567890@g.us", Text: "standup moved to 11", Timestamp: time.Now()}
	if err := SaveMessage(db, work); err != nil {
		t.Fatalf("same group message for a second account: %v", err)
	}
	if err := SaveContact(db, &Contact{Account: "work", JID: "919812345678@s.whatsapp.net", FullName: "Ravi (work)"}); err != nil {
		t.Fatalf("same contact for a second account: %v", err)
	}
	var maxLegacyID, workID int64
	if err := db.QueryRow("SELECT MAX(id) FROM whatsapp_messages WHERE account = 'default'").Scan(&maxLegacyID); err != nil {
		t.Fatalf("max legacy id: %v", err)
	}
	if err := db.QueryRow("SELECT id FROM whatsapp_messages WHERE account = 'work'").Scan(&workID); err != nil {
		t.Fatalf("work message id: %v", err)
	}
	if workID <= maxLegacyID {
		t.Fatalf("new message id %d collides with migrated ids up to %d", workID, maxLegacyID)
	}
	if contact, err := GetContact(db, DefaultAccount, "919812345678@s.whatsapp.net"); err != nil || contact.FullName != "Ravi Kumar" {
		t.Fatalf("default contact after work save = %+v, err %v", contact, err)
	}
}

func testMigrateConcurrent(t *testing.T, open dbFactory) {
	accounts := []string{"default", "work", "family", "assistant"}
	dbs := make([]*store.DB, len(accounts))
	for i := range dbs {
		dbs[i] = open(t)
	}

	start := make(chan struct{})
	errs := make(chan error, len(accounts))
	var wg sync.WaitGroup
	for i, account := range accounts {
		wg.Add(1)
		go func(db *store.DB, account string) {
			defer wg.Done()
			<-start
			if err := Migrate(db); err != nil {
				errs <- fmt.Errorf("%s migrate: %w", account, err)
				return
			}
			msg := &Message{Account: account, MessageID: "live-" + account, ChatJID: "919812345678@s.whatsapp.net", Text: "hi", Timestamp: time.Now()}
			if err := SaveMessage(db, msg); err != nil {
				errs <- fmt.Errorf("%s save: %w", account, err)
			}
		}(dbs[i], account)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if t.Failed() {
		return
	}

	db := open(t)
	if got := dump(t, db, "SELECT COUNT(*) FROM whatsapp_messages WHERE message_id LIKE '3EB0%' OR message_id LIKE 'BAE5%'"); got[0] != "4" {
		t.Fatalf("legacy messages after concurrent migrate = %v, want 4", got)
	}
	got := dump(t, db, "SELECT account, message_id FROM whatsapp_messages WHERE message_id LIKE 'live-%'")
	var want []string
	for _, a := range accounts {
		want = append(want, a+" live-"+a)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("live messages = %v, want %v", got, want)
	}
}

func TestMigrateLegacySQLite(t *testing.T)       { testMigrateLegacy(t, legacySQLite(t)) }
func TestMigrateLegacyPostgres(t *testing.T)     { testMigrateLegacy(t, legacyPostgres(t)) }
func TestMigrateConcurrentSQLite(t *testing.T)   { testMigrateConcurrent(t, legacySQLite(t)) }
func TestMigrateConcurrentPostgres(t *testing.T) { testMigrateConcurrent(t, legacyPostgres(t)) }
