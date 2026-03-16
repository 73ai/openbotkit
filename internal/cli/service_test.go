package cli

import "testing"

func TestResolveServices_NoArgs(t *testing.T) {
	names, err := resolveServices(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 || names[0] != "daemon" || names[1] != "server" {
		t.Fatalf("expected [daemon server], got %v", names)
	}
}

func TestResolveServices_Daemon(t *testing.T) {
	names, err := resolveServices([]string{"daemon"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "daemon" {
		t.Fatalf("expected [daemon], got %v", names)
	}
}

func TestResolveServices_Server(t *testing.T) {
	names, err := resolveServices([]string{"server"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "server" {
		t.Fatalf("expected [server], got %v", names)
	}
}

func TestResolveServices_Invalid(t *testing.T) {
	_, err := resolveServices([]string{"bogus"})
	if err == nil {
		t.Fatal("expected error for invalid service name")
	}
}

func TestServiceCmd_Subcommands(t *testing.T) {
	expected := map[string]bool{
		"run":       true,
		"install":   true,
		"uninstall": true,
		"start":     true,
		"stop":      true,
		"restart":   true,
		"status":    true,
		"logs":      true,
	}
	for _, sub := range serviceCmd.Commands() {
		delete(expected, sub.Name())
	}
	for name := range expected {
		t.Errorf("missing subcommand: %s", name)
	}
}
