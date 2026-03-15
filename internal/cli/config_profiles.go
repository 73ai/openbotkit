package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/spf13/cobra"
)

var configProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage model profiles",
}

var configProfilesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all model profiles (built-in + custom)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		activeProfile := ""
		if cfg.Models != nil {
			activeProfile = cfg.Models.Profile
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		// Built-in: singles first, then multis.
		fmt.Println("Built-in profiles:")
		fmt.Fprintln(w, "  \tNAME\tLABEL\tPROVIDERS")
		for _, name := range config.ProfileNames {
			p := config.Profiles[name]
			marker := " "
			if name == activeProfile {
				marker = "*"
			}
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", marker, name, p.Label, strings.Join(p.Providers, ", "))
		}
		w.Flush()

		// Custom profiles.
		if cfg.Models != nil && len(cfg.Models.CustomProfiles) > 0 {
			var names []string
			for n := range cfg.Models.CustomProfiles {
				names = append(names, n)
			}
			sort.Strings(names)

			fmt.Println("\nCustom profiles:")
			w = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  \tNAME\tLABEL\tPROVIDERS")
			for _, name := range names {
				cp := cfg.Models.CustomProfiles[name]
				marker := " "
				if name == activeProfile {
					marker = "*"
				}
				label := cp.Label
				if label == "" {
					label = name
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", marker, name, label, strings.Join(cp.Providers, ", "))
			}
			w.Flush()
		}

		return nil
	},
}

func init() {
	configProfilesCmd.AddCommand(configProfilesListCmd)
	configCmd.AddCommand(configProfilesCmd)
}
