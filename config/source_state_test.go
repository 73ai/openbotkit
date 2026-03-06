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
