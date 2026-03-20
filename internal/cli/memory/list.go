package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/service/memory"
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

		dir := config.UserMemoryDir()
		if err := memory.EnsureDir(dir); err != nil {
			return fmt.Errorf("ensure user_memory dir: %w", err)
		}

		s := memory.NewStore(dir)
		var memories []memory.Memory
		if listCategory != "" {
			memories, err = s.ListByCategory(memory.Category(listCategory))
		} else {
			memories, err = s.List()
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
