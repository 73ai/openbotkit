package service

import (
	"fmt"
	"os/exec"
	"strings"
)

type windowsManager struct {
	name string
}

func (m *windowsManager) taskName() string {
	return "OpenBotKit-" + m.name
}

func (m *windowsManager) Install(cfg *ServiceConfig) error {
	argsStr := strings.Join(cfg.Args, " ")
	return fmt.Errorf(
		"automatic service install is not supported on Windows\n\n"+
			"To run on startup, create a scheduled task:\n"+
			"  schtasks /create /tn \"%s\" /tr \"%s %s\" /sc onlogon /rl highest\n\n"+
			"Or run in foreground with: obk %s",
		m.taskName(), cfg.BinaryPath, argsStr, argsStr,
	)
}

func (m *windowsManager) Uninstall() error {
	return fmt.Errorf(
		"automatic service uninstall is not supported on Windows\n\n"+
			"To remove the scheduled task:\n"+
			"  schtasks /delete /tn \"%s\" /f",
		m.taskName(),
	)
}

func (m *windowsManager) Start() error {
	argsStr := strings.Join([]string{"obk"}, " ")
	return fmt.Errorf(
		"automatic service start is not supported on Windows\n\n"+
			"Run in foreground with: %s",
		argsStr,
	)
}

func (m *windowsManager) Stop() error {
	return fmt.Errorf(
		"automatic service stop is not supported on Windows\n\n" +
			"Stop the running process with Ctrl+C",
	)
}

func (m *windowsManager) Status() (string, error) {
	out, err := exec.Command("schtasks", "/query", "/tn", m.taskName()).Output()
	if err != nil {
		return "not installed", nil
	}
	if strings.Contains(string(out), "Running") {
		return "running", nil
	}
	return "installed (not running)", nil
}
