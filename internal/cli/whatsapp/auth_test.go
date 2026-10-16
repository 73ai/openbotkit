package whatsapp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/73ai/openbotkit/config"
	wasrc "github.com/73ai/openbotkit/source/whatsapp"
)

func TestLoginLocalSavesNewAccountOnlyAfterSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", dir)

	err := loginLocal(config.Default(), "work", "", "", func(wasrc.Config) error {
		return errors.New("whatsapp login failed (err-client-outdated)")
	})
	if err == nil || !strings.Contains(err.Error(), "err-client-outdated") {
		t.Fatalf("failed login err = %v", err)
	}
	if _, err := os.Stat(config.FilePath()); !os.IsNotExist(err) {
		t.Fatalf("config written after failed login: %v", err)
	}
	if config.IsWhatsAppAccountLinked("work") {
		t.Fatal("work linked after failed login")
	}

	var got wasrc.Config
	if err := loginLocal(config.Default(), "work", "", "", func(c wasrc.Config) error {
		got = c
		return nil
	}); err != nil {
		t.Fatalf("login: %v", err)
	}
	if got.Account != "work" || got.SessionDBPath != filepath.Join(dir, "whatsapp", "work", "session.db") || got.DataDSN != filepath.Join(dir, "whatsapp", "data.db") {
		t.Fatalf("login config = %+v", got)
	}
	if !config.IsWhatsAppAccountLinked("work") {
		t.Fatal("work not linked after login")
	}
	saved, err := config.LoadFrom(config.FilePath())
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if work := saved.WhatsApp.Accounts["work"]; work == nil || work.Role != "source" || saved.WhatsApp.Accounts["default"] == nil {
		t.Fatalf("saved accounts = %+v", saved.WhatsApp.Accounts)
	}
}

func TestLoginLocalRejectsInvalidAccountBeforeQR(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	for _, c := range [][3]string{{"assistant", "channel", ""}, {"work/personal", "", ""}, {"work", "reader", ""}} {
		err := loginLocal(config.Default(), c[0], c[1], c[2], func(wasrc.Config) error {
			t.Fatalf("QR login started for invalid input %v", c)
			return nil
		})
		if err == nil {
			t.Errorf("invalid input %v accepted", c)
		}
	}
}

func TestLoginLocalReloginKeepsConfigUntouched(t *testing.T) {
	t.Setenv("OBK_CONFIG_DIR", t.TempDir())
	cfg := config.Default()
	cfg.WhatsApp.Accounts = map[string]*config.WhatsAppAccount{
		"default":   {Role: "source"},
		"assistant": {Role: "channel", OwnerJID: "919876543210@s.whatsapp.net"},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	before, _ := os.ReadFile(config.FilePath())

	if err := loginLocal(cfg, "assistant", "", "", func(wasrc.Config) error { return nil }); err != nil {
		t.Fatalf("relogin: %v", err)
	}
	if after, _ := os.ReadFile(config.FilePath()); string(after) != string(before) {
		t.Fatalf("relogin rewrote config:\n%s", after)
	}
	if !config.IsWhatsAppAccountLinked("assistant") {
		t.Fatal("assistant not linked after relogin")
	}
}
