package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
)

func main() {
	seedFlag := flag.Bool("seed", false, "seed the database with sample use cases")
	flag.Parse()

	cfg := LoadConfig()

	db, err := openDB(cfg)
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		slog.Error("migrate database", "error", err)
		os.Exit(1)
	}

	st := NewStore(db)

	if *seedFlag {
		if err := Seed(st); err != nil {
			slog.Error("seed database", "error", err)
			os.Exit(1)
		}
		slog.Info("database seeded")
	}

	srv := NewServer(cfg, st)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
