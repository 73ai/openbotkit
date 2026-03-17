package applenotes

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	ansrc "github.com/73ai/openbotkit/source/applenotes"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "applenotes",
	Short: "Manage Apple Notes data source",
}

func init() {
	Cmd.AddCommand(syncCmd)
	Cmd.AddCommand(notesCmd)
}

func openAppleNotesDB(cfg *config.Config) (*store.DB, error) {
	if err := config.EnsureSourceDir("applenotes"); err != nil {
		return nil, fmt.Errorf("create applenotes dir: %w", err)
	}

	dsn := cfg.AppleNotesDataDSN()
	db, err := store.Open(store.Config{
		Driver: cfg.AppleNotes.Storage.Driver,
		DSN:    dsn,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := ansrc.Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}
