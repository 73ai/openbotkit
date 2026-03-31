package x

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "x",
	Short: "Manage X (formerly Twitter) data source",
}

func init() {
	Cmd.AddCommand(authCmd)
	Cmd.AddCommand(syncCmd)
	Cmd.AddCommand(timelineCmd)
	Cmd.AddCommand(tweetCmd)
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(notificationsCmd)
}
