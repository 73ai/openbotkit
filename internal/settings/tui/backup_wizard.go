package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/settings"
)

// --- Backup wizard: destination → credentials → verify → schedule → save ---

func (m model) enterBackupDest() (model, tea.Cmd) {
	m.state = stateBackupDest
	m.wizardError = ""

	dest := ""
	cfg := m.svc.Config()
	if cfg.Backup != nil && cfg.Backup.Destination != "" {
		dest = cfg.Backup.Destination
	}
	m.wizardBackupDest = &dest

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Where would you like to back up to?").
				Options(
					huh.NewOption("Cloudflare R2 (S3-compatible)", "r2"),
					huh.NewOption("Google Drive", "gdrive"),
				).
				Value(m.wizardBackupDest),
		),
	)
	return m, m.form.Init()
}

func (m model) updateBackupDest(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.state = stateBrowse
			m.form = nil
			m.wizardBackupDest = nil
			m.viewport.SetContent(m.renderTree())
			return m, nil
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		switch *m.wizardBackupDest {
		case "r2":
			return m.enterBackupR2Creds()
		case "gdrive":
			return m.enterBackupGDriveCreds()
		}
	}

	return m, cmd
}

func (m model) enterBackupR2Creds() (model, tea.Cmd) {
	m.state = stateBackupCreds
	m.wizardError = ""

	bucket := ""
	endpoint := ""
	ak := ""
	sk := ""

	cfg := m.svc.Config()
	if cfg.Backup != nil && cfg.Backup.R2 != nil {
		bucket = cfg.Backup.R2.Bucket
		endpoint = cfg.Backup.R2.Endpoint
	}

	m.wizardBackupBucket = &bucket
	m.wizardBackupEndpoint = &endpoint
	m.wizardBackupAK = &ak
	m.wizardBackupSK = &sk

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("R2 Bucket name").
				Value(m.wizardBackupBucket),
			huh.NewInput().
				Title("R2 Endpoint").
				Placeholder("https://<account-id>.r2.cloudflarestorage.com").
				Value(m.wizardBackupEndpoint),
			huh.NewInput().
				Title("Access Key ID").
				Value(m.wizardBackupAK),
			huh.NewInput().
				Title("Secret Access Key").
				EchoMode(huh.EchoModePassword).
				Value(m.wizardBackupSK),
		),
	)
	return m, m.form.Init()
}

func (m model) enterBackupGDriveCreds() (model, tea.Cmd) {
	m.state = stateBackupCreds
	m.wizardError = ""

	folder := ""
	cfg := m.svc.Config()
	if cfg.Backup != nil && cfg.Backup.GDrive != nil {
		folder = cfg.Backup.GDrive.FolderID
	}

	m.wizardBackupFolder = &folder

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Google Drive Folder ID").
				Description("Run 'obk setup' to configure Google OAuth and create the folder automatically").
				Value(m.wizardBackupFolder),
		),
	)
	return m, m.form.Init()
}

func (m model) updateBackupCreds(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.state = stateBrowse
			m.form = nil
			m.wizardBackupDest = nil
			m.viewport.SetContent(m.renderTree())
			return m, nil
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		switch *m.wizardBackupDest {
		case "r2":
			return m.handleR2CredsComplete()
		case "gdrive":
			return m.handleGDriveCredsComplete()
		}
	}

	return m, cmd
}

func (m model) handleR2CredsComplete() (model, tea.Cmd) {
	bucket := strings.TrimSpace(*m.wizardBackupBucket)
	endpoint := strings.TrimSpace(*m.wizardBackupEndpoint)
	ak := strings.TrimSpace(*m.wizardBackupAK)
	sk := strings.TrimSpace(*m.wizardBackupSK)

	if bucket == "" || endpoint == "" || ak == "" || sk == "" {
		m.wizardError = "All R2 fields are required"
		return m.enterBackupR2Creds()
	}

	// Store credentials.
	akRef := "keychain:obk/r2-access-key"
	skRef := "keychain:obk/r2-secret-key"

	if err := m.svc.StoreCredential(akRef, ak); err != nil {
		m.wizardError = fmt.Sprintf("Store access key: %v", err)
		return m.enterBackupR2Creds()
	}
	if err := m.svc.StoreCredential(skRef, sk); err != nil {
		m.wizardError = fmt.Sprintf("Store secret key: %v", err)
		return m.enterBackupR2Creds()
	}

	// Update config.
	cfg := m.svc.Config()
	if cfg.Backup == nil {
		cfg.Backup = &config.BackupConfig{}
	}
	cfg.Backup.Destination = "r2"
	if cfg.Backup.R2 == nil {
		cfg.Backup.R2 = &config.R2Config{}
	}
	cfg.Backup.R2.Bucket = bucket
	cfg.Backup.R2.Endpoint = endpoint
	cfg.Backup.R2.AccessKeyRef = akRef
	cfg.Backup.R2.SecretKeyRef = skRef

	// Verify connection.
	m.state = stateVerifying
	m.wizardError = ""
	return m, tea.Batch(
		m.wizardSpinner.Tick,
		verifyBackupCmd(m.svc, "r2"),
	)
}

func (m model) handleGDriveCredsComplete() (model, tea.Cmd) {
	folder := strings.TrimSpace(*m.wizardBackupFolder)
	if folder == "" {
		m.wizardError = "Folder ID is required"
		return m.enterBackupGDriveCreds()
	}

	// Update config.
	cfg := m.svc.Config()
	if cfg.Backup == nil {
		cfg.Backup = &config.BackupConfig{}
	}
	cfg.Backup.Destination = "gdrive"
	if cfg.Backup.GDrive == nil {
		cfg.Backup.GDrive = &config.GDriveConfig{}
	}
	cfg.Backup.GDrive.FolderID = folder

	// GDrive verification requires OAuth (complex); skip to schedule.
	return m.enterBackupSchedule()
}

func (m model) enterBackupSchedule() (model, tea.Cmd) {
	m.state = stateBackupSchedule
	m.wizardError = ""

	schedule := "6h"
	m.wizardBackupSchedule = &schedule

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How often should backups run?").
				Options(
					huh.NewOption("Every 6 hours", "6h"),
					huh.NewOption("Every 12 hours", "12h"),
					huh.NewOption("Daily", "24h"),
					huh.NewOption("Manual only", ""),
				).
				Value(m.wizardBackupSchedule),
		),
	)
	return m, m.form.Init()
}

func (m model) updateBackupSchedule(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.state = stateBrowse
			m.form = nil
			m.wizardBackupDest = nil
			m.viewport.SetContent(m.renderTree())
			return m, nil
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		return m.saveBackup()
	}

	return m, cmd
}

func (m model) saveBackup() (model, tea.Cmd) {
	cfg := m.svc.Config()
	cfg.Backup.Enabled = true
	cfg.Backup.Schedule = *m.wizardBackupSchedule

	if err := m.svc.Save(); err != nil {
		m.flash = fmt.Sprintf("Error saving: %v", err)
	} else {
		m.flash = "Backup configured and enabled!"
	}

	m.state = stateBrowse
	m.form = nil
	m.wizardBackupDest = nil
	m.svc.RebuildTree()
	m.rebuildRows()
	m.viewport.SetContent(m.renderTree())
	return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return flashMsg{}
	})
}

func verifyBackupCmd(svc *settings.Service, dest string) tea.Cmd {
	return func() tea.Msg {
		err := svc.VerifyBackup(dest, svc.Config())
		return backupVerifyResultMsg{err: err}
	}
}
