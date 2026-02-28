package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type DB struct {
	*sql.DB
	driver string
}

type Config struct {
	Driver string // "sqlite" or "postgres"
	DSN    string
}

func Open(cfg Config) (*DB, error) {
	switch cfg.Driver {
	case "sqlite":
		return openSQLite(cfg.DSN)
	case "postgres":
		return openPostgres(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}

func SQLiteConfig(path string) Config {
	return Config{Driver: "sqlite", DSN: path}
}

func PostgresConfig(dsn string) Config {
	return Config{Driver: "postgres", DSN: dsn}
}

func (db *DB) IsSQLite() bool {
	return db.driver == "sqlite"
}

func (db *DB) IsPostgres() bool {
	return db.driver == "postgres"
}

// Rebind rewrites ? placeholders to $1, $2, ... for Postgres.
// For SQLite, the query is returned unchanged.
func (db *DB) Rebind(query string) string {
	if db.IsSQLite() {
		return query
	}
	var buf strings.Builder
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			buf.WriteString("$")
			buf.WriteString(strconv.Itoa(n))
			n++
		} else {
			buf.WriteByte(query[i])
		}
	}
	return buf.String()
}
