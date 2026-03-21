package backup

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	backupsvc "github.com/73ai/openbotkit/service/backup"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show last backup info",
	RunE: func(cmd *cobra.Command, args []string) error {
		manifest, err := backupsvc.LoadManifest(config.BackupLastManifestPath())
		if err != nil {
			return fmt.Errorf("load last manifest: %w", err)
		}

		if manifest.ID == "" {
			fmt.Println("No backup has been run yet.")
			fmt.Println("Run: obk backup now")
			return nil
		}

		fmt.Printf("Last backup: %s\n", manifest.ID)
		fmt.Printf("Timestamp:   %s\n", manifest.Timestamp.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("Hostname:    %s\n", manifest.Hostname)
		fmt.Printf("Files:       %d\n", len(manifest.Files))

		var totalSize, totalCompressed int64
		for _, f := range manifest.Files {
			totalSize += f.Size
			totalCompressed += f.CompressedSize
		}
		fmt.Printf("Total size:  %s (compressed: %s)\n", formatBytes(totalSize), formatBytes(totalCompressed))
		return nil
	},
}
