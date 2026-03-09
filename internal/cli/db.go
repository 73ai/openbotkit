package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/spf13/cobra"
)

var dbCmd = &cobra.Command{
	Use:   "db <source> <sql>",
	Short: "Run a SQL query against a data source",
	Long:  "Execute a read-only SQL query against one of the data sources (gmail, whatsapp, history, user_memory, applenotes).",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		query := args[1]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.IsRemote() {
			return fmt.Errorf("remote mode not yet implemented for db command")
		}

		return dbLocal(cfg, source, query)
	},
}

func dbLocal(cfg *config.Config, source, query string) error {
	dsn, err := cfg.SourceDataDSN(source)
	if err != nil {
		return err
	}

	if _, err := os.Stat(dsn); err != nil {
		return fmt.Errorf("database not found: %s", dsn)
	}

	sqlite3, err := exec.LookPath("sqlite3")
	if err != nil {
		return fmt.Errorf("sqlite3 not found in PATH — install it to use this command")
	}

	c := exec.Command(sqlite3, "-header", "-separator", "\t", dsn, query)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func init() {
	rootCmd.AddCommand(dbCmd)
}
