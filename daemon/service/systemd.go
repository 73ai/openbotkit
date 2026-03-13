package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const unitTemplate = `[Unit]
Description=OpenBotKit %s
After=network.target

[Service]
Type=simple
ExecStart=%s %s
%sRestart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

type systemdManager struct {
	name string
}

func (m *systemdManager) unitName() string {
	return "obk-" + m.name
}

func (m *systemdManager) unitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "systemd", "user", m.unitName()+".service"), nil
}

func (m *systemdManager) Install(cfg *ServiceConfig) error {
	path, err := m.unitPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}

	argsStr := strings.Join(cfg.Args, " ")

	var envLines string
	for k, v := range cfg.Env {
		envLines += fmt.Sprintf("Environment=%s=%s\n", k, v)
	}

	content := fmt.Sprintf(unitTemplate, cfg.Name, cfg.BinaryPath, argsStr, envLines)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	unit := m.unitName()
	if err := exec.Command("systemctl", "--user", "enable", "--now", unit).Run(); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}

	return nil
}

func (m *systemdManager) Start() error {
	if err := exec.Command("systemctl", "--user", "start", m.unitName()).Run(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	return nil
}

func (m *systemdManager) Stop() error {
	if err := exec.Command("systemctl", "--user", "stop", m.unitName()).Run(); err != nil {
		return fmt.Errorf("stop service: %w", err)
	}
	return nil
}

func (m *systemdManager) Uninstall() error {
	unit := m.unitName()
	_ = exec.Command("systemctl", "--user", "disable", "--now", unit).Run()

	path, err := m.unitPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file: %w", err)
	}

	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()

	return nil
}

func (m *systemdManager) Status() (string, error) {
	out, err := exec.Command("systemctl", "--user", "is-active", m.unitName()).Output()
	if err != nil {
		return "not running", nil
	}
	return strings.TrimSpace(string(out)), nil
}
