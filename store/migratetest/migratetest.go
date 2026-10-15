package migratetest

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/73ai/openbotkit/store"
)

type Fixture struct {
	Name     string
	SQLite   string
	Postgres string
	Preserve []string
}

type Suite struct {
	Source   string
	Migrate  func(*store.DB) error
	Steps    []store.Migration
	Fixtures []Fixture
	Use      func(db *store.DB, worker int) error
	Check    func(t *testing.T, db *store.DB, workers int)
}

const Workers = 4

type opener func(t *testing.T) *store.DB

func Run(t *testing.T, s Suite) {
	t.Run("steps", func(t *testing.T) { checkSteps(t, s.Steps) })
	fixtures := append([]Fixture{{Name: "empty"}}, s.Fixtures...)
	for _, driver := range []string{"sqlite", "postgres"} {
		t.Run(driver, func(t *testing.T) {
			newDB := seedSQLite
			if driver == "postgres" {
				newDB = postgres(t)
			}
			var fresh []string
			for _, fx := range fixtures {
				t.Run(fx.Name, func(t *testing.T) {
					schema := runSequential(t, s, fx, newDB(t, fx))
					if fresh == nil {
						fresh = schema
					} else if !reflect.DeepEqual(schema, fresh) {
						t.Fatalf("schema migrated from %s differs from a fresh database\ngot:  %v\nwant: %v", fx.Name, schema, fresh)
					}
				})
				t.Run(fx.Name+"_concurrent", func(t *testing.T) { runConcurrent(t, s, fx, newDB(t, fx)) })
			}
		})
	}
}

func checkSteps(t *testing.T, steps []store.Migration) {
	for i, step := range steps {
		if step.Version != i+1 {
			t.Errorf("step %d has version %d: versions must count up from 1 with no gaps or reuse", i, step.Version)
		}
		if hasSQL := step.SQLite != "" && step.Postgres != ""; hasSQL == (step.Func != nil) {
			t.Errorf("version %d must have either SQL for both drivers or a Func", step.Version)
		}
	}
}

func runSequential(t *testing.T, s Suite, fx Fixture, open opener) []string {
	db := open(t)
	before := dumpAll(t, db, fx.Preserve)
	migrate(t, s, db)
	schema := snapshot(t, db)
	applied := appliedVersions(t, db, s)
	migrate(t, s, db)
	if got := snapshot(t, db); !reflect.DeepEqual(got, schema) {
		t.Fatalf("second migrate changed schema\nbefore: %v\nafter:  %v", schema, got)
	}
	if got := appliedVersions(t, db, s); !reflect.DeepEqual(got, applied) {
		t.Fatalf("second migrate changed schema_migrations\nbefore: %v\nafter:  %v", applied, got)
	}
	if after := dumpAll(t, db, fx.Preserve); !reflect.DeepEqual(after, before) {
		t.Fatalf("rows changed by migration\nbefore: %v\nafter:  %v", before, after)
	}
	use(t, s, db, 0)
	if s.Check != nil {
		s.Check(t, db, 1)
	}
	return schema
}

func runConcurrent(t *testing.T, s Suite, fx Fixture, open opener) {
	before := dumpAll(t, open(t), fx.Preserve)
	dbs := make([]*store.DB, Workers)
	for i := range dbs {
		dbs[i] = open(t)
	}
	start := make(chan struct{})
	errs := make(chan error, Workers)
	var wg sync.WaitGroup
	for i, db := range dbs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := s.Migrate(db); err != nil {
				errs <- fmt.Errorf("worker %d migrate: %w", i, err)
				return
			}
			if s.Use != nil {
				if err := s.Use(db, i); err != nil {
					errs <- fmt.Errorf("worker %d use: %w", i, err)
				}
			}
		}()
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
	appliedVersions(t, db, s)
	after := dumpAll(t, db, fx.Preserve)
	for i := range before {
		if missing := subtract(before[i], after[i]); len(missing) > 0 {
			t.Fatalf("rows lost or changed by concurrent migration: %v", missing)
		}
	}
	if s.Check != nil {
		s.Check(t, db, Workers)
	}
}

func migrate(t *testing.T, s Suite, db *store.DB) {
	t.Helper()
	if err := s.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func use(t *testing.T, s Suite, db *store.DB, worker int) {
	t.Helper()
	if s.Use == nil {
		return
	}
	if err := s.Use(db, worker); err != nil {
		t.Fatalf("use after migrate: %v", err)
	}
}

func appliedVersions(t *testing.T, db *store.DB, s Suite) []string {
	t.Helper()
	rows := Dump(t, db, fmt.Sprintf("SELECT version, applied_at FROM schema_migrations WHERE source = '%s' ORDER BY version", s.Source))
	var got, want []string
	for _, row := range rows {
		got = append(got, strings.Fields(row)[0])
	}
	for _, step := range s.Steps {
		want = append(want, strconv.Itoa(step.Version))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema_migrations versions for %s = %v, want %v", s.Source, got, want)
	}
	return rows
}

func subtract(rows, from []string) []string {
	left := map[string]int{}
	for _, r := range from {
		left[r]++
	}
	var missing []string
	for _, r := range rows {
		if left[r] == 0 {
			missing = append(missing, r)
		}
		left[r]--
	}
	return missing
}

func dumpAll(t *testing.T, db *store.DB, queries []string) [][]string {
	var out [][]string
	for _, q := range queries {
		out = append(out, Dump(t, db, q))
	}
	return out
}

func snapshot(t *testing.T, db *store.DB) []string {
	t.Helper()
	if db.IsPostgres() {
		return append(
			Dump(t, db, "SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = current_schema() ORDER BY 1, 2"),
			Dump(t, db, "SELECT tablename, indexname, indexdef FROM pg_indexes WHERE schemaname = current_schema() ORDER BY 1, 2")...)
	}
	return append(
		Dump(t, db, "SELECT m.name, p.name FROM sqlite_master m JOIN pragma_table_info(m.name) p WHERE m.type = 'table' ORDER BY 1, 2"),
		Dump(t, db, "SELECT m.tbl_name, m.name, i.seqno, i.name FROM sqlite_master m JOIN pragma_index_info(m.name) i WHERE m.type = 'index' ORDER BY 1, 2, 3")...)
}

func Dump(t *testing.T, db *store.DB, query string) []string {
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

func openDB(t *testing.T, cfg store.Config) *store.DB {
	db, err := store.Open(cfg)
	if err != nil {
		t.Fatalf("open %s: %v", cfg.Driver, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func seed(t *testing.T, db *store.DB, name, sql string) {
	if sql == "" {
		return
	}
	if _, err := db.Exec(sql); err != nil {
		t.Fatalf("seed %s: %v", name, err)
	}
}

func seedSQLite(t *testing.T, fx Fixture) opener {
	cfg := store.SQLiteConfig(filepath.Join(t.TempDir(), "data.db"))
	seed(t, openDB(t, cfg), fx.Name, fx.SQLite)
	return func(t *testing.T) *store.DB { return openDB(t, cfg) }
}

func postgres(t *testing.T) func(t *testing.T, fx Fixture) opener {
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
	dsn := func(name string) store.Config {
		return store.PostgresConfig(fmt.Sprintf("postgres://postgres:obk@%s:%s/%s?sslmode=disable", host, port.Port(), name))
	}
	admin := openDB(t, dsn("obk"))
	n := 0
	return func(t *testing.T, fx Fixture) opener {
		n++
		name := fmt.Sprintf("m%d", n)
		if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
			t.Fatalf("create database: %v", err)
		}
		cfg := dsn(name)
		seed(t, openDB(t, cfg), fx.Name, fx.Postgres)
		return func(t *testing.T) *store.DB { return openDB(t, cfg) }
	}
}
