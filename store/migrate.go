package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrSchemaTooNew = errors.New("update obk")

type Migration struct {
	Version  int
	SQLite   string
	Postgres string
	Func     func(ctx context.Context, conn *sql.Conn, db *DB) error
}

const migrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	source TEXT NOT NULL,
	version INTEGER NOT NULL,
	applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(source, version)
)`

func (db *DB) Migrate(source string, steps []Migration) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	begin := []string{"PRAGMA busy_timeout=30000", "BEGIN IMMEDIATE"}
	if db.IsPostgres() {
		begin = []string{"BEGIN", "SELECT pg_advisory_xact_lock(hashtext('obk_schema_migrations'))"}
	}
	for _, stmt := range append(begin, migrationsTable) {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	defer conn.ExecContext(ctx, "ROLLBACK")

	applied := map[int]bool{}
	rows, err := conn.QueryContext(ctx, db.Rebind("SELECT version FROM schema_migrations WHERE source = ?"), source)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	known := 0
	for _, step := range steps {
		known = max(known, step.Version)
	}
	for v := range applied {
		if v > known {
			return fmt.Errorf("database %s is at schema version %d but this obk only knows up to %d; %w", source, v, known, ErrSchemaTooNew)
		}
	}

	for _, step := range steps {
		if applied[step.Version] {
			continue
		}
		if err := step.apply(ctx, conn, db); err != nil {
			return fmt.Errorf("%s migration %d: %w", source, step.Version, err)
		}
		if _, err := conn.ExecContext(ctx, db.Rebind("INSERT INTO schema_migrations (source, version) VALUES (?, ?)"), source, step.Version); err != nil {
			return err
		}
	}
	_, err = conn.ExecContext(ctx, "COMMIT")
	return err
}

func (m Migration) apply(ctx context.Context, conn *sql.Conn, db *DB) error {
	if m.Func != nil {
		return m.Func(ctx, conn, db)
	}
	stmt := m.SQLite
	if db.IsPostgres() {
		stmt = m.Postgres
	}
	_, err := conn.ExecContext(ctx, stmt)
	return err
}

func HasColumn(ctx context.Context, conn *sql.Conn, db *DB, table, column string) (bool, error) {
	query := "SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?"
	if db.IsPostgres() {
		query = "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2"
	}
	var n int
	err := conn.QueryRowContext(ctx, query, table, column).Scan(&n)
	return n > 0, err
}
