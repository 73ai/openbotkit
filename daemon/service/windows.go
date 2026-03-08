package service

import (
	"fmt"
	"os/exec"
	"strings"
)

type windowsManager struct{}

func (m *windowsManager) Install(cfg *ServiceConfig) error {
	return fmt.Errorf(
		"automatic service install is not supported on Windows\n\n"+
			"To run on startup, create a scheduled task:\n"+
			"  schtasks /create /tn \"OpenBotKit\" /tr \"%s service run\" /sc onlogon /rl highest\n\n"+
			"Or run in foreground with: obk service run",
		cfg.BinaryPath,
	)
}

func (m *windowsManager) Uninstall() error {
	return fmt.Errorf(
		"automatic service uninstall is not supported on Windows\n\n" +
			"To remove the scheduled task:\n" +
			"  schtasks /delete /tn \"OpenBotKit\" /f",
	)
}

func (m *windowsManager) Start() error {
	return fmt.Errorf(
		"automatic service start is not supported on Windows\n\n" +
			"Run in foreground with: obk service run",
	)
}

func (m *windowsManager) Stop() error {
	return fmt.Errorf(
		"automatic service stop is not supported on Windows\n\n" +
			"Stop the running process with Ctrl+C",
	)
}

func (m *windowsManager) Status() (string, error) {
	out, err := exec.Command("schtasks", "/query", "/tn", "OpenBotKit").Output()
	if err != nil {
		return "not installed", nil
	}
	if strings.Contains(string(out), "Running") {
		return "running", nil
	}
	return "installed (not running)", nil
}
