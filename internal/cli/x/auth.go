package x

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/73ai/openbotkit/agent/audit"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/internal/tty"
	"github.com/73ai/openbotkit/source/twitter/client"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage X authentication",
	RunE:  authInteractiveRun,
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

		return authInteractiveRun(cmd, args)
	},
}

func authInteractiveRun(cmd *cobra.Command, args []string) error {
	if err := tty.RequireInteractive("obk x auth login --token <token>"); err != nil {
		return err
	}

	var username, password string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username, email, or phone").
				Value(&username),
			huh.NewInput().
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&password),
		),
	).Run()
	if err != nil {
		return err
	}

	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	fmt.Println("Signing in to X...")
	logXAudit("x.auth.login", "username="+username, "attempting login", "")

	result, err := client.Login(username, password)
	if err != nil {
		logXAudit("x.auth.login", "username="+username, "", err.Error())
		return fmt.Errorf("login failed: %w", err)
	}

	if result.NeedsTFA {
		var code string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Verification code").
					Description("Enter the code from your authenticator app").
					Value(&code),
			),
		).Run()
		if err != nil {
			return err
		}

		code = strings.TrimSpace(code)
		if code == "" {
			return fmt.Errorf("verification code is required")
		}

		fmt.Println("Verifying...")
		logXAudit("x.auth.login_tfa", "username="+username, "submitting 2FA code", "")
		result, err = client.LoginWithTFA(username, password, code)
		if err != nil {
			logXAudit("x.auth.login_tfa", "username="+username, "", err.Error())
			return fmt.Errorf("2FA verification failed: %w", err)
		}
	}

	if result.Session == nil {
		logXAudit("x.auth.login", "username="+username, "", "no session returned")
		return fmt.Errorf("login failed: no session returned")
	}

	if err := client.SaveSession(result.Session); err != nil {
		logXAudit("x.auth.login", "username="+username, "", err.Error())
		return fmt.Errorf("save credentials: %w", err)
	}

	if err := config.LinkSource("x"); err != nil {
		return fmt.Errorf("link source: %w", err)
	}

	logXAudit("x.auth.login", "username="+username, "login successful", "")
	fmt.Println("Signed in to X successfully!")
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
