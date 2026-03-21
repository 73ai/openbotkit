package backup

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage backups",
}

func init() {
	Cmd.AddCommand(nowCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(restoreCmd)
}
