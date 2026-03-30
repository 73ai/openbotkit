package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/73ai/openbotkit/agent/audit"
	"github.com/73ai/openbotkit/config"
	xclient "github.com/73ai/openbotkit/source/twitter/client"
)

// --- X login wizard: username/password → spinner → optional 2FA → save ---

func (m model) enterXLogin() (model, tea.Cmd) {
	m.state = stateXAuth

	username := ""
	password := ""
	if m.wizardXUsername != nil {
		username = *m.wizardXUsername
	}
	if m.wizardXPassword != nil {
		password = *m.wizardXPassword
	}
	m.wizardXUsername = &username
	m.wizardXPassword = &password

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Username, email, or phone").
				Value(m.wizardXUsername),
			huh.NewInput().
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(m.wizardXPassword),
		),
	)
	return m, m.form.Init()
}

func (m model) updateXAuth(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m.exitXWizard("")
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		username := strings.TrimSpace(*m.wizardXUsername)
		password := strings.TrimSpace(*m.wizardXPassword)

		if username == "" || password == "" {
			m.wizardError = "Username and password are required"
			return m.enterXLogin()
		}

		m.state = stateVerifying
		m.wizardError = ""
		return m, tea.Batch(
			m.wizardSpinner.Tick,
			xLoginCmd(username, password),
		)
	}

	return m, cmd
}

func (m model) enterXTFA() (model, tea.Cmd) {
	m.state = stateXTFA

	code := ""
	m.wizardXTFACode = &code

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Verification code").
				Description("Enter the code from your authenticator app").
				Value(m.wizardXTFACode),
		),
	)
	return m, m.form.Init()
}

func (m model) updateXTFA(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m.exitXWizard("")
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		code := strings.TrimSpace(*m.wizardXTFACode)
		if code == "" {
			m.wizardError = "Verification code is required"
			return m.enterXTFA()
		}

		username := *m.wizardXUsername
		password := *m.wizardXPassword

		m.state = stateVerifying
		m.wizardError = ""
		return m, tea.Batch(
			m.wizardSpinner.Tick,
			xLoginTFACmd(username, password, code),
		)
	}

	return m, cmd
}

func (m model) handleXLoginResult(msg xAuthResultMsg) (model, tea.Cmd) {
	if msg.err != nil {
		m.wizardError = fmt.Sprintf("Login failed: %v", msg.err)
		return m.enterXLogin()
	}

	if msg.needsTFA {
		return m.enterXTFA()
	}

	if msg.session == nil {
		m.wizardError = "Login failed: no session returned"
		return m.enterXLogin()
	}

	if err := xclient.SaveSession(msg.session); err != nil {
		return m.exitXWizard(fmt.Sprintf("Save credentials failed: %v", err))
	}

	cfg := m.svc.Config()
	if cfg.X == nil {
		cfg.X = &config.XConfig{}
	}

	if err := m.svc.Save(); err != nil {
		return m.exitXWizard(fmt.Sprintf("Save config failed: %v", err))
	}

	if err := config.LinkSource("x"); err != nil {
		return m.exitXWizard(fmt.Sprintf("Link source failed: %v", err))
	}

	username := ""
	if msg.session.Username != "" {
		username = " as @" + msg.session.Username
	}

	return m.exitXWizard(fmt.Sprintf("Connected to X%s!", username))
}

func (m model) exitXWizard(flash string) (model, tea.Cmd) {
	m.state = stateBrowse
	m.form = nil
	m.wizardXUsername = nil
	m.wizardXPassword = nil
	m.wizardXTFACode = nil
	m.wizardError = ""
	m.svc.RebuildTree()
	m.rebuildRows()
	m.viewport.SetContent(m.renderTree())

	if flash != "" {
		m.flash = flash
		m.viewport.SetContent(m.renderTree())
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return flashMsg{}
		})
	}
	return m, nil
}

func logXAudit(toolName, input, output, errMsg string) {
	l := audit.OpenDefault(config.AuditJSONLPath())
	if l == nil {
		return
	}
	defer l.Close()
	l.Log(audit.Entry{
		Context:      "settings",
		ToolName:     toolName,
		InputSummary: input,
		OutputSummary: output,
		Error:        errMsg,
	})
}

func xLoginCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		logXAudit("x.auth.login", "username="+username, "attempting login", "")

		result, err := xclient.Login(username, password)
		if err != nil {
			logXAudit("x.auth.login", "username="+username, "", err.Error())
			return xAuthResultMsg{err: err}
		}
		if result.NeedsTFA {
			logXAudit("x.auth.login", "username="+username, "2FA required", "")
			return xAuthResultMsg{needsTFA: true}
		}

		logXAudit("x.auth.login", "username="+username, "login successful", "")
		return xAuthResultMsg{session: result.Session}
	}
}

func xLoginTFACmd(username, password, code string) tea.Cmd {
	return func() tea.Msg {
		logXAudit("x.auth.login_tfa", "username="+username, "submitting 2FA code", "")

		result, err := xclient.LoginWithTFA(username, password, code)
		if err != nil {
			logXAudit("x.auth.login_tfa", "username="+username, "", err.Error())
			return xAuthResultMsg{err: err}
		}

		logXAudit("x.auth.login_tfa", "username="+username, "2FA login successful", "")
		return xAuthResultMsg{session: result.Session}
	}
}
