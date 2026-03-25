package config

import (
	"os"
	"testing"
)

func TestSourceState(t *testing.T) {
	tmp := t.TempDir()
	os.Setenv("OBK_CONFIG_DIR", tmp)
	t.Cleanup(func() { os.Unsetenv("OBK_CONFIG_DIR") })

	// Not linked by default
	if IsSourceLinked("testapp") {
		t.Fatal("expected not linked initially")
	}

	// Link it
	if err := LinkSource("testapp"); err != nil {
		t.Fatalf("link: %v", err)
	}
	if !IsSourceLinked("testapp") {
		t.Fatal("expected linked after LinkSource")
	}

	// Verify state persists via LoadSourceState
	state, err := LoadSourceState("testapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !state.Linked {
		t.Fatal("expected Linked=true from LoadSourceState")
	}

	// Unlink it
	if err := UnlinkSource("testapp"); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if IsSourceLinked("testapp") {
		t.Fatal("expected not linked after UnlinkSource")
	}
}

func TestIsWhatsAppAccountLinked_FalseWhenNoFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	if IsWhatsAppAccountLinked("myaccount") {
		t.Fatal("expected not linked when no file exists")
	}
}

func TestLinkWhatsAppAccount_CreatesStateFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	if err := LinkWhatsAppAccount("myaccount"); err != nil {
		t.Fatalf("link: %v", err)
	}
	if !IsWhatsAppAccountLinked("myaccount") {
		t.Fatal("expected linked after LinkWhatsAppAccount")
	}
}

func TestUnlinkWhatsAppAccount_RemovesState(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	if err := LinkWhatsAppAccount("myaccount"); err != nil {
		t.Fatalf("link: %v", err)
	}
	if err := UnlinkWhatsAppAccount("myaccount"); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if IsWhatsAppAccountLinked("myaccount") {
		t.Fatal("expected not linked after unlink")
	}
}

func TestMultipleAccounts_IndependentState(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	if err := LinkWhatsAppAccount("accountA"); err != nil {
		t.Fatalf("link A: %v", err)
	}
	if IsWhatsAppAccountLinked("accountB") {
		t.Fatal("accountB should not be linked")
	}
	if !IsWhatsAppAccountLinked("accountA") {
		t.Fatal("accountA should be linked")
	}
}

func TestWhatsAppAccountLinked_DefaultDelegatesToSource(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OBK_CONFIG_DIR", tmp)

	if err := LinkWhatsAppAccount("default"); err != nil {
		t.Fatalf("link default: %v", err)
	}
	if !IsSourceLinked("whatsapp") {
		t.Fatal("default label should delegate to IsSourceLinked")
	}
	if !IsWhatsAppAccountLinked("default") {
		t.Fatal("expected default linked")
	}
}
