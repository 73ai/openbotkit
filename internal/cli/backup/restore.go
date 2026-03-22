package backup

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	backupsvc "github.com/73ai/openbotkit/service/backup"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id>",
	Short: "Restore from a backup snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshotID := args[0]

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

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			manifest, err := svc.GetManifest(ctx, snapshotID)
			if err != nil {
				return fmt.Errorf("fetch manifest: %w", err)
			}
			fmt.Printf("This will overwrite %d files in %s\n", len(manifest.Files), config.Dir())
			fmt.Print("Continue? [y/N] ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		result, err := svc.Restore(ctx, snapshotID)
		if err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}

		fmt.Printf("Restored %d files from snapshot %s\n", result.Restored, snapshotID)
		return nil
	},
}

func init() {
	restoreCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
