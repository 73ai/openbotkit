package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/73ai/openbotkit/agent/audit"
	"github.com/73ai/openbotkit/config"
	xclient "github.com/73ai/openbotkit/source/twitter/client"
)

// --- X login wizard: extract cookies from browser → spinner → save ---

func (m model) enterXLogin() (model, tea.Cmd) {
	m.state = stateVerifying
	m.wizardError = ""
	m.wizardXBrowser = true
	return m, tea.Batch(
		m.wizardSpinner.Tick,
		xExtractCookiesCmd(),
	)
}

func (m model) handleXLoginResult(msg xAuthResultMsg) (model, tea.Cmd) {
	if msg.err != nil {
		errMsg := fmt.Sprintf("No X session found: %v", msg.err)
		errMsg += "\nSign in to x.com in your browser first, then try again."
		return m.exitXWizard(errMsg)
	}

	if msg.session == nil {
		return m.exitXWizard("Cookie extraction failed: no session returned")
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

	browser := ""
	if msg.browser != "" {
		browser = " (from " + msg.browser + ")"
	}

	return m.exitXWizard(fmt.Sprintf("Connected to X%s!", browser))
}

func (m model) exitXWizard(flash string) (model, tea.Cmd) {
	m.state = stateBrowse
	m.form = nil
	m.wizardXBrowser = false
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
		Context:       "settings",
		ToolName:      toolName,
		InputSummary:  input,
		OutputSummary: output,
		Error:         errMsg,
	})
}

func xExtractCookiesCmd() tea.Cmd {
	return func() tea.Msg {
		logXAudit("x.auth.login_browser", "", "extracting cookies from browser", "")

		session, browser, err := xclient.ExtractSessionFromBrowser()
		if err != nil {
			logXAudit("x.auth.login_browser", "", "", err.Error())
			return xAuthResultMsg{err: err}
		}

		logXAudit("x.auth.login_browser", "", "browser login successful ("+browser+")", "")
		return xAuthResultMsg{session: session, browser: browser}
	}
}
