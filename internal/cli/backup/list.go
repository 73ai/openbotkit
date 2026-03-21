package backup

import (
	"fmt"

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
			fmt.Println(id)
		}
		return nil
	},
}
