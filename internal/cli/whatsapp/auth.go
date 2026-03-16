package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/priyanshujain/openbotkit/config"
	wasrc "github.com/priyanshujain/openbotkit/source/whatsapp"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage WhatsApp authentication",
}

var authLoginCmd = &cobra.Command{
	Use:     "login",
	Short:   "Authenticate WhatsApp by scanning a QR code",
	Example: `  obk whatsapp auth login`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.IsRemote() {
			return authLoginRemote(cfg)
		}

		if err := config.EnsureSourceDir("whatsapp"); err != nil {
			return fmt.Errorf("create whatsapp dir: %w", err)
		}

		w := wasrc.New(wasrc.Config{
			SessionDBPath: cfg.WhatsAppSessionDBPath(),
			DataDSN:       cfg.WhatsAppDataDSN(),
		})

		ctx := context.Background()
		if err := w.Login(ctx); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := config.LinkSource("whatsapp"); err != nil {
			return fmt.Errorf("link source: %w", err)
		}

		fmt.Println("\nSuccessfully authenticated WhatsApp")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Disconnect and clear WhatsApp session",
	Example: `  obk whatsapp auth logout
  obk whatsapp auth logout --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Print("About to disconnect WhatsApp session. Continue? (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		w := wasrc.New(wasrc.Config{
			SessionDBPath: cfg.WhatsAppSessionDBPath(),
		})

		ctx := context.Background()
		if err := w.Logout(ctx); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}

		if err := config.UnlinkSource("whatsapp"); err != nil {
			return fmt.Errorf("unlink source: %w", err)
		}

		fmt.Println("Logged out of WhatsApp")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "list",
	Short: "List WhatsApp authentication status",
	Example: `  obk whatsapp auth list
  obk whatsapp auth list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		jsonOut, _ := cmd.Flags().GetBool("json")

		ctx := context.Background()
		client, err := wasrc.NewClient(ctx, cfg.WhatsAppSessionDBPath())
		if err != nil {
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]any{"authenticated": false})
			}
			fmt.Println("Not authenticated (no session found).")
			return nil
		}
		defer client.Disconnect()

		if !client.IsAuthenticated() {
			if jsonOut {
				return json.NewEncoder(os.Stdout).Encode(map[string]any{"authenticated": false})
			}
			fmt.Println("Not authenticated.")
			return nil
		}

		user := client.WM().Store.ID.User
		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"authenticated": true, "user": user})
		}
		fmt.Printf("Authenticated as %s\n", user)
		return nil
	},
}

func authLoginRemote(cfg *config.Config) error {
	if cfg.Remote == nil || cfg.Remote.Server == "" {
		return fmt.Errorf("remote server not configured — run 'obk setup' to configure")
	}

	url := strings.TrimRight(cfg.Remote.Server, "/") + "/auth/whatsapp"
	fmt.Printf("Open this URL in your browser to authenticate WhatsApp:\n%s\n", url)

	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", url).Start()
	}

	client, err := newRemoteClient(cfg)
	if err != nil {
		return err
	}
	fmt.Println("\nWaiting for authentication to complete...")
	if err := client.WaitWhatsAppAuth(); err != nil {
		return err
	}
	fmt.Println("WhatsApp authenticated successfully!")
	return nil
}

func init() {
	authLogoutCmd.Flags().Bool("force", false, "Skip confirmation prompt")
	authStatusCmd.Flags().Bool("json", false, "Output as JSON")
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
