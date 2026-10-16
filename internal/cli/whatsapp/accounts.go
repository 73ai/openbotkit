package whatsapp

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/73ai/openbotkit/config"
	wasrc "github.com/73ai/openbotkit/source/whatsapp"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage WhatsApp accounts",
}

var accountsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List configured WhatsApp accounts with their sync status",
	Example: `  obk whatsapp accounts list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if err := config.EnsureSourceDir("whatsapp"); err != nil {
			return fmt.Errorf("create whatsapp dir: %w", err)
		}
		db, err := store.Open(store.Config{Driver: cfg.WhatsApp.Storage.Driver, DSN: cfg.WhatsAppDataDSN()})
		if err != nil {
			return fmt.Errorf("open whatsapp db: %w", err)
		}
		defer db.Close()
		if err := wasrc.Migrate(db); err != nil {
			return fmt.Errorf("migrate whatsapp db: %w", err)
		}
		stats, err := wasrc.StatsByAccount(db)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "LABEL\tROLE\tLINKED\tMESSAGES\tLAST SYNC\tOWNER")
		for _, e := range cfg.WhatsAppAccountList() {
			linked, lastSync, owner := "no", "never", "-"
			if config.IsWhatsAppAccountLinked(e.Label) {
				linked = "yes"
			}
			s := stats[e.Label]
			if s.LastSyncedAt != nil {
				lastSync = s.LastSyncedAt.Local().Format("2006-01-02 15:04")
			}
			if e.OwnerJID != "" {
				owner = e.OwnerJID
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", e.Label, e.Role, linked, s.Messages, lastSync, owner)
		}
		return w.Flush()
	},
}

func init() {
	accountsCmd.AddCommand(accountsListCmd)
}
