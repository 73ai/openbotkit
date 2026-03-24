package whatsapp

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage WhatsApp accounts",
}

var accountsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List configured WhatsApp accounts",
	Example: `  obk whatsapp accounts list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		entries := cfg.WhatsAppAccountList()
		if len(entries) == 0 {
			fmt.Println("No accounts configured.")
			return nil
		}

		for _, e := range entries {
			linked := "not linked"
			if config.IsWhatsAppAccountLinked(e.Label) {
				linked = "linked"
			}
			fmt.Printf("%-15s role=%-8s %s", e.Label, e.Role, linked)
			if e.OwnerJID != "" {
				fmt.Printf("  owner=%s", e.OwnerJID)
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	accountsCmd.AddCommand(accountsListCmd)
}
