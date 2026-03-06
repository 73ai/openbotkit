package applenotes

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/priyanshujain/openbotkit/config"
	ansrc "github.com/priyanshujain/openbotkit/source/applenotes"
	"github.com/spf13/cobra"
)

var notesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Query stored Apple Notes",
}

var notesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored notes with optional filters",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openAppleNotesDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		folder, _ := cmd.Flags().GetString("folder")
		limit, _ := cmd.Flags().GetInt("limit")
		jsonOut, _ := cmd.Flags().GetBool("json")

		notes, err := ansrc.ListNotes(db, ansrc.ListOptions{
			Folder: folder,
			Limit:  limit,
		})
		if err != nil {
			return fmt.Errorf("list notes: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(notes)
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "APPLE ID\tFOLDER\tMODIFIED\tTITLE")
		for _, n := range notes {
			title := n.Title
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			folder := n.Folder
			if len(folder) > 20 {
				folder = folder[:17] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				truncateID(n.AppleID), folder,
				n.ModifiedAt.Format("2006-01-02"), title)
		}
		return w.Flush()
	},
}

var notesGetCmd = &cobra.Command{
	Use:   "get <apple-id>",
	Short: "Show full details of a stored note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openAppleNotesDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		jsonOut, _ := cmd.Flags().GetBool("json")

		note, err := ansrc.GetNote(db, args[0])
		if err != nil {
			return fmt.Errorf("get note: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(note)
		}

		fmt.Printf("Apple ID:  %s\n", note.AppleID)
		fmt.Printf("Title:     %s\n", note.Title)
		fmt.Printf("Folder:    %s\n", note.Folder)
		fmt.Printf("Account:   %s\n", note.Account)
		fmt.Printf("Created:   %s\n", note.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Modified:  %s\n", note.ModifiedAt.Format("2006-01-02 15:04:05"))
		if note.PasswordProtected {
			fmt.Printf("Password:  protected\n")
		}
		fmt.Printf("\n--- Body ---\n%s\n", note.Body)
		return nil
	},
}

var notesSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search across title and body",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		db, err := openAppleNotesDB(cfg)
		if err != nil {
			return err
		}
		defer db.Close()

		jsonOut, _ := cmd.Flags().GetBool("json")
		limit, _ := cmd.Flags().GetInt("limit")

		notes, err := ansrc.SearchNotes(db, args[0], limit)
		if err != nil {
			return fmt.Errorf("search notes: %w", err)
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(notes)
		}

		if len(notes) == 0 {
			fmt.Println("No notes found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "APPLE ID\tFOLDER\tMODIFIED\tTITLE")
		for _, n := range notes {
			title := n.Title
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				truncateID(n.AppleID), n.Folder,
				n.ModifiedAt.Format("2006-01-02"), title)
		}
		return w.Flush()
	},
}

func init() {
	notesListCmd.Flags().String("folder", "", "Filter by folder name")
	notesListCmd.Flags().Int("limit", 50, "Maximum number of results")
	notesListCmd.Flags().Bool("json", false, "Output as JSON")

	notesGetCmd.Flags().Bool("json", false, "Output as JSON")

	notesSearchCmd.Flags().Bool("json", false, "Output as JSON")
	notesSearchCmd.Flags().Int("limit", 50, "Maximum number of results")

	notesCmd.AddCommand(notesListCmd)
	notesCmd.AddCommand(notesGetCmd)
	notesCmd.AddCommand(notesSearchCmd)
}

// truncateID shortens a CoreData URI for display.
func truncateID(id string) string {
	if len(id) > 30 {
		return "..." + id[len(id)-25:]
	}
	return id
}
