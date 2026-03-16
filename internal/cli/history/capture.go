package history

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/priyanshujain/openbotkit/config"
	historysrc "github.com/priyanshujain/openbotkit/source/history"
	"github.com/priyanshujain/openbotkit/store"
	"github.com/spf13/cobra"
)

var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture a conversation from a Claude Code transcript",
	Long:  "Reads capture input as JSON from stdin. Designed to be called by Claude Code hooks.",
	Example: `  echo '{"session_id":"abc","transcript_path":"/tmp/t.json"}' | obk history capture`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var input historysrc.CaptureInput
		if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
			return fmt.Errorf("decode stdin: %w", err)
		}

		if input.SessionID == "" || input.TranscriptPath == "" {
			return fmt.Errorf("session_id and transcript_path are required")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if err := config.EnsureSourceDir("history"); err != nil {
			return fmt.Errorf("ensure history dir: %w", err)
		}

		dsn := cfg.HistoryDataDSN()
		db, err := store.Open(store.Config{
			Driver: cfg.History.Storage.Driver,
			DSN:    dsn,
		})
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		if err := historysrc.Capture(db, input); err != nil {
			return fmt.Errorf("capture: %w", err)
		}

		return nil
	},
}
