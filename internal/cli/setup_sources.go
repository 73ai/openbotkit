package cli

import (
	"fmt"
	"runtime"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/spf13/cobra"
)

var setupAppleContactsCmd = &cobra.Command{
	Use:   "applecontacts",
	Short: "Set up Apple Contacts integration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("Apple Contacts is only available on macOS")
		}
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
		if runtime.GOOS != "darwin" {
			return fmt.Errorf("Apple Notes is only available on macOS")
		}
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
