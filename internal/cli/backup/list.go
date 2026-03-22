package backup

import (
	"fmt"
	"strings"
	"time"

	"github.com/73ai/openbotkit/config"
	backupsvc "github.com/73ai/openbotkit/service/backup"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backup snapshots",
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
		snapshots, err := svc.ListSnapshots(ctx)
		if err != nil {
			return fmt.Errorf("list snapshots: %w", err)
		}

		if len(snapshots) == 0 {
			fmt.Println("No snapshots found.")
			return nil
		}

		for _, id := range snapshots {
			fmt.Printf("  %s  %s\n", formatSnapshotDate(id), id)
		}
		return nil
	},
}

func formatSnapshotDate(id string) string {
	// ID format: 20060102T150405Z-<hex>
	ts := id
	if idx := strings.Index(id, "-"); idx > 0 {
		ts = id[:idx]
	}
	t, err := time.Parse("20060102T150405Z", ts)
	if err != nil {
		return "                   "
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
