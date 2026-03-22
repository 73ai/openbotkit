package backup

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	backupsvc "github.com/73ai/openbotkit/service/backup"
	"github.com/spf13/cobra"
)

var nowCmd = &cobra.Command{
	Use:   "now",
	Short: "Run a backup immediately",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		ctx := cmd.Context()
		backend, err := resolveBackend(ctx, cfg)
		if err != nil {
			return err
		}

		svc := backupsvc.New(backend, config.Dir())
		result, err := svc.Run(ctx)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}

		fmt.Printf("Backup complete: %d changed, %d unchanged, %s uploaded in %s\n",
			result.Changed, result.Skipped, formatBytes(result.Uploaded), result.Duration.Round(100*1e6))
		return nil
	},
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
