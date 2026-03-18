package cli

import (
	"bytes"
	"strings"
	"testing"

	clicli "github.com/73ai/openbotkit/channel/cli"
)

func TestCLIInteractor_Notify(t *testing.T) {
	var out bytes.Buffer
	ch := clicli.New(strings.NewReader(""), &out)
	inter := NewCLIInteractor(ch)
	if err := inter.Notify("hello"); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if !strings.Contains(out.String(), "hello") {
		t.Errorf("output = %q, want to contain hello", out.String())
	}
}

func TestCLIInteractor_RequestApproval_Yes(t *testing.T) {
	var out bytes.Buffer
	ch := clicli.New(strings.NewReader("y\n"), &out)
	inter := NewCLIInteractor(ch)
	ok, err := inter.RequestApproval("run dangerous command")
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	if !ok {
		t.Error("expected approval to be granted")
	}
}

func TestCLIInteractor_RequestApproval_No(t *testing.T) {
	var out bytes.Buffer
	ch := clicli.New(strings.NewReader("n\n"), &out)
	inter := NewCLIInteractor(ch)
	ok, err := inter.RequestApproval("run dangerous command")
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	if ok {
		t.Error("expected approval to be denied")
	}
}

func TestCLIInteractor_RequestApproval_DefaultDeny(t *testing.T) {
	var out bytes.Buffer
	ch := clicli.New(strings.NewReader("\n"), &out)
	inter := NewCLIInteractor(ch)
	ok, err := inter.RequestApproval("run dangerous command")
	if err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	if ok {
		t.Error("expected default to deny")
	}
}

func TestCLIInteractor_NotifyLink(t *testing.T) {
	var out bytes.Buffer
	ch := clicli.New(strings.NewReader(""), &out)
	inter := NewCLIInteractor(ch)
	if err := inter.NotifyLink("Click here", "https://example.com"); err != nil {
		t.Fatalf("NotifyLink: %v", err)
	}
	if !strings.Contains(out.String(), "https://example.com") {
		t.Errorf("output = %q, want to contain URL", out.String())
	}
}
