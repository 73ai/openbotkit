package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
%s
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>WatchPaths</key>
	<array>
		<string>%s</string>
	</array>
%s
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>`

type launchdManager struct {
	name string
}

func (m *launchdManager) label() string {
	return "com.openbotkit.obk." + m.name
}

func (m *launchdManager) plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", m.label()+".plist"), nil
}

func (m *launchdManager) Install(cfg *ServiceConfig) error {
	path, err := m.plistPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cfg.LogPath), 0700); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	var argLines []string
	argLines = append(argLines, fmt.Sprintf("\t\t<string>%s</string>", cfg.BinaryPath))
	for _, a := range cfg.Args {
		argLines = append(argLines, fmt.Sprintf("\t\t<string>%s</string>", a))
	}
	argsXML := strings.Join(argLines, "\n")

	var envXML string
	if len(cfg.Env) > 0 {
		var envLines []string
		envLines = append(envLines, "\t<key>EnvironmentVariables</key>")
		envLines = append(envLines, "\t<dict>")
		for k, v := range cfg.Env {
			envLines = append(envLines, fmt.Sprintf("\t\t<key>%s</key>", k))
			envLines = append(envLines, fmt.Sprintf("\t\t<string>%s</string>", v))
		}
		envLines = append(envLines, "\t</dict>")
		envXML = strings.Join(envLines, "\n")
	}

	content := fmt.Sprintf(plistTemplate, m.label(), argsXML, cfg.BinaryPath, envXML, cfg.LogPath, cfg.LogPath)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	if err := exec.Command("launchctl", "load", path).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	return nil
}

func (m *launchdManager) Start() error {
	path, err := m.plistPath()
	if err != nil {
		return err
	}

	if err := exec.Command("launchctl", "load", path).Run(); err != nil {
		return fmt.Errorf("launchctl load: %w", err)
	}

	return nil
}

func (m *launchdManager) Stop() error {
	path, err := m.plistPath()
	if err != nil {
		return err
	}

	if err := exec.Command("launchctl", "unload", path).Run(); err != nil {
		return fmt.Errorf("launchctl unload: %w", err)
	}

	return nil
}

func (m *launchdManager) Uninstall() error {
	path, err := m.plistPath()
	if err != nil {
		return err
	}

	_ = exec.Command("launchctl", "unload", path).Run()

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}

	return nil
}

func (m *launchdManager) Status() (string, error) {
	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return "unknown", fmt.Errorf("launchctl list: %w", err)
	}

	label := m.label()
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, label) {
			return "running", nil
		}
	}

	return "not running", nil
}
