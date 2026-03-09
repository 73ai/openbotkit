package auth

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/remote"
	wasrc "github.com/priyanshujain/openbotkit/source/whatsapp"
	"github.com/spf13/cobra"
)

var whatsappCmd = &cobra.Command{
	Use:   "whatsapp",
	Short: "Manage WhatsApp authentication",
}

var whatsappLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate WhatsApp by scanning a QR code",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if cfg.IsRemote() {
			return whatsappLoginRemote(cfg)
		}

		if err := config.EnsureSourceDir("whatsapp"); err != nil {
			return fmt.Errorf("create whatsapp dir: %w", err)
		}

		w := wasrc.New(wasrc.Config{
			SessionDBPath: cfg.WhatsAppSessionDBPath(),
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

var whatsappLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Disconnect and clear WhatsApp session",
	RunE: func(cmd *cobra.Command, args []string) error {
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

var whatsappStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show WhatsApp authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		ctx := context.Background()
		client, err := wasrc.NewClient(ctx, cfg.WhatsAppSessionDBPath())
		if err != nil {
			fmt.Println("Not authenticated (no session found).")
			return nil
		}
		defer client.Disconnect()

		if !client.IsAuthenticated() {
			fmt.Println("Not authenticated.")
			return nil
		}

		fmt.Printf("Authenticated as %s\n", client.WM().Store.ID.User)
		return nil
	},
}

func whatsappLoginRemote(cfg *config.Config) error {
	if cfg.Remote == nil || cfg.Remote.Server == "" {
		return fmt.Errorf("remote server not configured — run 'obk setup' to configure")
	}

	url := strings.TrimRight(cfg.Remote.Server, "/") + "/auth/whatsapp"
	fmt.Printf("Open this URL in your browser to authenticate WhatsApp:\n%s\n", url)

	// Try to open the browser automatically.
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", url).Start()
	}

	// Poll the server until authentication completes.
	client := remote.NewClient(cfg.Remote.Server, cfg.Remote.Username, cfg.Remote.Password)
	fmt.Println("\nWaiting for authentication to complete...")
	if err := client.WaitWhatsAppAuth(); err != nil {
		return err
	}
	fmt.Println("WhatsApp authenticated successfully!")
	return nil
}

func init() {
	whatsappCmd.AddCommand(whatsappLoginCmd)
	whatsappCmd.AddCommand(whatsappLogoutCmd)
	whatsappCmd.AddCommand(whatsappStatusCmd)
}
