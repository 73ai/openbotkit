package whatsapp

import (
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/spf13/cobra"
)

func loginFlags(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("role", "source", "")
	cmd.Flags().String("owner-jid", "", "")
	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return cmd
}

func TestAccountEntryToRegisterNewAccount(t *testing.T) {
	entry := accountEntryToRegister(loginFlags(t), config.Default(), "work")
	if entry == nil || entry.Role != "source" {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestAccountEntryToRegisterSkipsConfiguredAccount(t *testing.T) {
	if entry := accountEntryToRegister(loginFlags(t), config.Default(), "default"); entry != nil {
		t.Fatalf("implicit default re-registered: %+v", entry)
	}

	cfg := config.Default()
	cfg.WhatsApp.Accounts = map[string]*config.WhatsAppAccount{"bot": {Role: "channel", OwnerJID: "1@s.whatsapp.net"}}
	if entry := accountEntryToRegister(loginFlags(t), cfg, "bot"); entry != nil {
		t.Fatalf("re-login without flags changed config: %+v", entry)
	}
}

func TestAccountEntryToRegisterAppliesFlagsOverExisting(t *testing.T) {
	cfg := config.Default()
	cfg.WhatsApp.Accounts = map[string]*config.WhatsAppAccount{"bot": {Role: "channel", OwnerJID: "1@s.whatsapp.net"}}
	entry := accountEntryToRegister(loginFlags(t, "--role", "both"), cfg, "bot")
	if entry == nil || entry.Role != "both" || entry.OwnerJID != "1@s.whatsapp.net" {
		t.Fatalf("entry = %+v", entry)
	}
}
