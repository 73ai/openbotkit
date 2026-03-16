package cli

import (
	"github.com/priyanshujain/openbotkit/config"
	"github.com/spf13/cobra"
)

var setupAppleContactsCmd = &cobra.Command{
	Use:   "applecontacts",
	Short: "Set up Apple Contacts integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return setupAppleContacts(cfg)
	},
}

var setupAppleNotesCmd = &cobra.Command{
	Use:   "applenotes",
	Short: "Set up Apple Notes integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return setupAppleNotes(cfg)
	},
}

func init() {
	setupCmd.AddCommand(setupAppleContactsCmd)
	setupCmd.AddCommand(setupAppleNotesCmd)
}
