package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/memory"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var listCategory string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List personal memories",
	Example: `  obk memory list
  obk memory list --category identity
  obk memory list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		jsonOut, _ := cmd.Flags().GetBool("json")

		if cfg.IsRemote() {
			client, err := newRemoteClient(cfg)
			if err != nil {
				return err
			}
			items, err := client.MemoryList(listCategory)
			if err != nil {
				return fmt.Errorf("list: %w", err)
			}
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(items)
			}
			if len(items) == 0 {
				fmt.Println("No memories stored.")
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tCATEGORY\tCONTENT\tSOURCE")
			for _, m := range items {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", m.ID, m.Category, m.Content, m.Source)
			}
			return w.Flush()
		}

		if err := config.EnsureSourceDir("user_memory"); err != nil {
			return fmt.Errorf("ensure user_memory dir: %w", err)
		}

		db, err := store.Open(store.Config{
			Driver: cfg.UserMemory.Storage.Driver,
			DSN:    cfg.UserMemoryDataDSN(),
		})
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer db.Close()

		if err := memory.Migrate(db); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}

		var memories []memory.Memory
		if listCategory != "" {
			memories, err = memory.ListByCategory(db, memory.Category(listCategory))
		} else {
			memories, err = memory.List(db)
		}
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(memories)
		}

		if len(memories) == 0 {
			fmt.Println("No memories stored.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tCATEGORY\tCONTENT\tSOURCE")
		for _, m := range memories {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", m.ID, m.Category, m.Content, m.Source)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().StringVar(&listCategory, "category", "", "filter by category (identity, preference, relationship, project)")
	listCmd.Flags().Bool("json", false, "Output as JSON")
}
