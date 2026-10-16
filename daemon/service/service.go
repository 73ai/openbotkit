package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Platform string

const (
	PlatformMacOS   Platform = "macos"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
	PlatformUnknown Platform = "unknown"
)

type ServiceConfig struct {
	Name       string // service name, e.g. "daemon" or "server"
	BinaryPath string
	Args       []string // command arguments, e.g. ["service", "run"]
	LogPath    string
	Env        map[string]string // environment variables to set
}

func DetectPlatform() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMacOS
	case "linux":
		return PlatformLinux
	case "windows":
		return PlatformWindows
	default:
		return PlatformUnknown
	}
}

func DefaultConfig(name string, args []string) (*ServiceConfig, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find obk binary: %w", err)
	}
	binPath, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return nil, fmt.Errorf("resolve binary path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	return &ServiceConfig{
		Name:       name,
		BinaryPath: binPath,
		Args:       args,
		LogPath:    filepath.Join(home, ".obk", name+".log"),
		Env: map[string]string{
			"PATH": os.Getenv("PATH"),
			"HOME": home,
		},
	}, nil
}

type Manager interface {
	Install(cfg *ServiceConfig) error
	Installed() bool
	Uninstall() error
	Start() error
	Stop() error
	Status() (string, error)
}

func NewManager(name string) (Manager, error) {
	switch DetectPlatform() {
	case PlatformMacOS:
		return &launchdManager{name: name}, nil
	case PlatformLinux:
		return &systemdManager{name: name}, nil
	case PlatformWindows:
		return &windowsManager{name: name}, nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
