package service

import (
	"runtime"
	"strings"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	p := DetectPlatform()

	switch runtime.GOOS {
	case "darwin":
		if p != PlatformMacOS {
			t.Errorf("expected PlatformMacOS on darwin, got %s", p)
		}
	case "linux":
		if p != PlatformLinux {
			t.Errorf("expected PlatformLinux on linux, got %s", p)
		}
	case "windows":
		if p != PlatformWindows {
			t.Errorf("expected PlatformWindows on windows, got %s", p)
		}
	default:
		if p != PlatformUnknown {
			t.Errorf("expected PlatformUnknown on %s, got %s", runtime.GOOS, p)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg, err := DefaultConfig("daemon", []string{"service", "run"})
	if err != nil {
		t.Fatalf("DefaultConfig failed: %v", err)
	}

	if cfg.Name != "daemon" {
		t.Errorf("Name = %q, want %q", cfg.Name, "daemon")
	}
	if cfg.BinaryPath == "" {
		t.Error("BinaryPath is empty")
	}
	if cfg.LogPath == "" {
		t.Error("LogPath is empty")
	}
	if !strings.HasSuffix(cfg.LogPath, "daemon.log") {
		t.Errorf("LogPath %q should end with daemon.log", cfg.LogPath)
	}
}

func TestNewManager(t *testing.T) {
	mgr, err := NewManager("daemon")

	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		if err != nil {
			t.Fatalf("NewManager failed on %s: %v", runtime.GOOS, err)
		}
		if mgr == nil {
			t.Fatal("NewManager returned nil")
		}
	default:
		if err == nil {
			t.Error("NewManager should fail on unsupported platform")
		}
	}
}
