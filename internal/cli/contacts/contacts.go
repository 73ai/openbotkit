package contacts

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/73ai/openbotkit/config"
	contactsrc "github.com/73ai/openbotkit/source/contacts"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage unified contacts",
}

func init() {
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(syncCmd)

	searchCmd.Flags().Int("limit", 20, "Maximum number of results")
	searchCmd.Flags().Bool("json", false, "Output as JSON")
	getCmd.Flags().Bool("json", false, "Output as JSON")
	listCmd.Flags().Int("limit", 50, "Maximum number of results")
	listCmd.Flags().Bool("json", false, "Output as JSON")
}

func openContactsDB(cfg *config.Config) (*store.DB, error) {
	if err := config.EnsureSourceDir("contacts"); err != nil {
		return nil, fmt.Errorf("create contacts dir: %w", err)
	}

	db, err := store.Open(store.Config{
		Driver: cfg.Contacts.Storage.Driver,
		DSN:    cfg.ContactsDataDSN(),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := contactsrc.Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search contacts by name",
	Example: `  obk contacts search "John"
  obk contacts search "Jane Doe" --limit 10 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openContactsDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		limit, _ := cmd.Flags().GetInt("limit")
		jsonOut, _ := cmd.Flags().GetBool("json")

		results, err := contactsrc.SearchContacts(db, args[0], limit)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(results)
		}

		if len(results) == 0 {
			fmt.Println("No contacts found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tMATCHED\tIDENTITIES\tMESSAGES")
		for _, r := range results {
			identities := ""
			for _, id := range r.Contact.Identities {
				if identities != "" {
					identities += ", "
				}
				identities += id.IdentityType + ":" + id.IdentityValue
			}
			totalMsgs := 0
			for _, inter := range r.Contact.Interactions {
				totalMsgs += inter.MessageCount
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n",
				r.Contact.ID, r.Contact.DisplayName, r.MatchedAlias,
				truncate(identities, 60), totalMsgs)
		}
		return w.Flush()
	},
}

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show full contact details",
	Example: `  obk contacts get 42
  obk contacts get 42 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openContactsDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}

		jsonOut, _ := cmd.Flags().GetBool("json")

		contact, err := contactsrc.GetContact(db, id)
		if err != nil {
			return fmt.Errorf("get contact: %w", err)
		}
		if contact == nil {
			return fmt.Errorf("contact %d not found", id)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(contact)
		}

		fmt.Printf("ID:       %d\n", contact.ID)
		fmt.Printf("Name:     %s\n", contact.DisplayName)
		fmt.Printf("Created:  %s\n", contact.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:  %s\n", contact.UpdatedAt.Format("2006-01-02 15:04:05"))

		if len(contact.Identities) > 0 {
			fmt.Println("\nIdentities:")
			for _, id := range contact.Identities {
				fmt.Printf("  [%s] %s: %s", id.Source, id.IdentityType, id.IdentityValue)
				if id.DisplayName != "" {
					fmt.Printf(" (%s)", id.DisplayName)
				}
				fmt.Println()
			}
		}

		if len(contact.Aliases) > 0 {
			fmt.Printf("\nAliases: %s\n", joinStrings(contact.Aliases))
		}

		if len(contact.Interactions) > 0 {
			fmt.Println("\nInteractions:")
			for _, inter := range contact.Interactions {
				fmt.Printf("  %s: %d messages", inter.Channel, inter.MessageCount)
				if inter.LastInteractionAt != nil {
					fmt.Printf(" (last: %s)", inter.LastInteractionAt.Format("2006-01-02"))
				}
				fmt.Println()
			}
		}

		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List contacts",
	Example: `  obk contacts list
  obk contacts list --limit 100 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openContactsDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		limit, _ := cmd.Flags().GetInt("limit")
		jsonOut, _ := cmd.Flags().GetBool("json")

		contacts, err := contactsrc.ListContacts(db, limit, 0)
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(contacts)
		}

		if len(contacts) == 0 {
			fmt.Println("No contacts found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME")
		for _, c := range contacts {
			fmt.Fprintf(w, "%d\t%s\n", c.ID, c.DisplayName)
		}
		return w.Flush()
	},
}

var syncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "Sync contacts from all linked sources",
	Example: `  obk contacts sync`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openContactsDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		sourceDBs := make(map[string]*store.DB)

		sources := []struct {
			name   string
			driver string
			dsn    string
		}{
			{"whatsapp", cfg.WhatsApp.Storage.Driver, cfg.WhatsAppDataDSN()},
			{"gmail", cfg.Gmail.Storage.Driver, cfg.GmailDataDSN()},
			{"imessage", cfg.IMessage.Storage.Driver, cfg.IMessageDataDSN()},
		}
		for _, s := range sources {
			if !config.IsSourceLinked(s.name) {
				continue
			}
			sdb, err := store.Open(store.Config{Driver: s.driver, DSN: s.dsn})
			if err != nil {
				fmt.Printf("  warning: could not open %s db: %v\n", s.name, err)
				continue
			}
			defer sdb.Close()
			sourceDBs[s.name] = sdb
		}

		result, err := contactsrc.Sync(db, sourceDBs, contactsrc.SyncOptions{})
		if err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		fmt.Printf("\nSync complete: %d created, %d linked", result.Created, result.Linked)
		if result.Errors > 0 {
			fmt.Printf(", %d errors", result.Errors)
		}
		fmt.Println()
		return nil
	},
}

func truncate(s string, max int) string {
	if max < 4 || len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func joinStrings(ss []string) string {
	return strings.Join(ss, ", ")
}
