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

var configProfilesShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a model profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Check built-in profiles first.
		if p, ok := config.Profiles[name]; ok {
			category := p.Category
			if category == "single" {
				category = "single (1 API key)"
			} else {
				category = fmt.Sprintf("multi (%d API keys)", len(p.Providers))
			}
			fmt.Printf("Name:        %s\n", p.Name)
			fmt.Printf("Label:       %s\n", p.Label)
			fmt.Printf("Description: %s\n", p.Description)
			fmt.Printf("Category:    %s\n", category)
			fmt.Printf("Providers:   %s\n", strings.Join(p.Providers, ", "))
			fmt.Println()
			fmt.Println("Tiers:")
			fmt.Printf("  Default: %s\n", p.Tiers.Default)
			fmt.Printf("  Complex: %s\n", p.Tiers.Complex)
			fmt.Printf("  Fast:    %s\n", p.Tiers.Fast)
			fmt.Printf("  Nano:    %s\n", p.Tiers.Nano)
			return nil
		}

		// Check custom profiles.
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.Models != nil {
			if cp, ok := cfg.Models.CustomProfiles[name]; ok {
				label := cp.Label
				if label == "" {
					label = name
				}
				fmt.Printf("Name:        %s\n", name)
				fmt.Printf("Label:       %s\n", label)
				if cp.Description != "" {
					fmt.Printf("Description: %s\n", cp.Description)
				}
				fmt.Printf("Category:    custom\n")
				fmt.Printf("Providers:   %s\n", strings.Join(cp.Providers, ", "))
				fmt.Println()
				fmt.Println("Tiers:")
				fmt.Printf("  Default: %s\n", cp.Tiers.Default)
				fmt.Printf("  Complex: %s\n", cp.Tiers.Complex)
				fmt.Printf("  Fast:    %s\n", cp.Tiers.Fast)
				fmt.Printf("  Nano:    %s\n", cp.Tiers.Nano)
				return nil
			}
		}

		return fmt.Errorf("profile %q not found", name)
	},
}

func init() {
	configProfilesCmd.AddCommand(configProfilesListCmd)
	configProfilesCmd.AddCommand(configProfilesShowCmd)
	configCmd.AddCommand(configProfilesCmd)
}
