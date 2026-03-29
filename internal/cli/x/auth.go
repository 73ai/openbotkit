package x

import (
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/source/twitter/client"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage X authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with X using auth_token from browser",
	Long: `To get your auth_token:
  1. Open x.com in Chrome and log in
  2. Open DevTools (F12) → Application → Cookies → x.com
  3. Copy the value of "auth_token"
  4. Run: obk x auth login --token <paste>`,
	Example: "  obk x auth login --token abc123def456",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _ := cmd.Flags().GetString("token")
		token = strings.TrimSpace(token)
		if token == "" {
			return fmt.Errorf("--token is required (paste auth_token from browser cookies)")
		}

		session := client.NewSession(token)

		if err := client.SaveSession(session); err != nil {
			return fmt.Errorf("save credentials: %w", err)
		}

		fmt.Println("Authenticated with X successfully.")
		fmt.Println("Run 'obk x sync' to fetch your timeline.")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear X credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.DeleteSession(); err != nil {
			return fmt.Errorf("delete credentials: %w", err)
		}
		fmt.Println("X credentials removed.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check X authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := client.LoadSession()
		if err != nil {
			fmt.Println("Not authenticated. Run 'obk x auth login' to connect.")
			return nil
		}
		fmt.Println("Authenticated with X.")
		if session.Username != "" {
			fmt.Printf("Username: @%s\n", session.Username)
		}
		return nil
	},
}

func init() {
	authLoginCmd.Flags().String("token", "", "auth_token value from browser cookies")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
