package applenotes

import (
	"testing"

	"github.com/73ai/openbotkit/store/migratetest"
)

func TestMigrations(t *testing.T) {
	migratetest.Run(t, migratetest.Suite{
		Source:   "applenotes",
		Migrate:  Migrate,
		Steps:    migrations,
		Fixtures: []migratetest.Fixture{{Name: "pre_runner", SQLite: schemaSQLite, Postgres: schemaPostgres}},
	})
}
