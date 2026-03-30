package x

import (
	"fmt"
	"strings"

	"github.com/73ai/openbotkit/agent/audit"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/twitter/client"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage X authentication",
	RunE:  authBrowserRun,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to X",
	Example: `  obk x auth login
  obk x auth login --token <browser-cookie-token>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _ := cmd.Flags().GetString("token")
		token = strings.TrimSpace(token)

		if token != "" {
			logXAudit("x.auth.login_token", "", "token-based login", "")
			session := client.NewSession(token)
			if err := client.SaveSession(session); err != nil {
				logXAudit("x.auth.login_token", "", "", err.Error())
				return fmt.Errorf("save credentials: %w", err)
			}
			if err := config.LinkSource("x"); err != nil {
				return fmt.Errorf("link source: %w", err)
			}
			logXAudit("x.auth.login_token", "", "token login successful", "")
			fmt.Println("Authenticated with X successfully.")
			fmt.Println("Run 'obk x sync' to fetch your timeline.")
			return nil
		}

		return authBrowserRun(cmd, args)
	},
}

func authBrowserRun(cmd *cobra.Command, args []string) error {
	fmt.Println("Checking browsers for X session...")
	logXAudit("x.auth.login_browser", "", "extracting cookies from browser", "")

	session, browser, err := client.ExtractSessionFromBrowser()
	if err != nil {
		logXAudit("x.auth.login_browser", "", "", err.Error())
		fmt.Println("No X session found in any browser.")
		fmt.Println("Sign in to x.com in your browser first, then try again.")
		fmt.Println("Or use: obk x auth login --token <token>")
		return fmt.Errorf("browser cookie extraction failed: %w", err)
	}

	if err := client.SaveSession(session); err != nil {
		logXAudit("x.auth.login_browser", "", "", err.Error())
		return fmt.Errorf("save credentials: %w", err)
	}

	if err := config.LinkSource("x"); err != nil {
		return fmt.Errorf("link source: %w", err)
	}

	logXAudit("x.auth.login_browser", "", "browser login successful ("+browser+")", "")
	fmt.Printf("Authenticated with X (from %s).\n", browser)
	fmt.Println("Run 'obk x sync' to fetch your timeline.")
	return nil
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out of X",
	RunE: func(cmd *cobra.Command, args []string) error {
		logXAudit("x.auth.logout", "", "signing out", "")
		if err := client.DeleteSession(); err != nil {
			logXAudit("x.auth.logout", "", "", err.Error())
			return fmt.Errorf("delete credentials: %w", err)
		}
		logXAudit("x.auth.logout", "", "signed out", "")
		fmt.Println("Signed out of X.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check X authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := client.LoadSession()
		if err != nil {
			fmt.Println("Not signed in. Run 'obk x auth login' to connect.")
			return nil
		}
		fmt.Println("Signed in to X.")
		if session.Username != "" {
			fmt.Printf("Account: @%s\n", session.Username)
		}
		return nil
	},
}

func logXAudit(toolName, input, output, errMsg string) {
	l := audit.OpenDefault(config.AuditJSONLPath())
	if l == nil {
		return
	}
	defer l.Close()
	l.Log(audit.Entry{
		Context:       "cli",
		ToolName:      toolName,
		InputSummary:  input,
		OutputSummary: output,
		Error:         errMsg,
	})
}

func init() {
	authLoginCmd.Flags().String("token", "", "auth_token from browser cookies (advanced)")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
