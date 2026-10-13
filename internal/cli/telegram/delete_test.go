package telegram

import (
	"strings"
	"testing"
)

func TestDeleteRejectsAgents(t *testing.T) {
	t.Setenv("OBK_AGENT", "1")
	for _, cmd := range []func() error{
		func() error { return runDelete(chatsDeleteCmd, true) },
		func() error { return runDelete(messagesDeleteCmd, false) },
	} {
		err := cmd()
		if err == nil || !strings.Contains(err.Error(), "manual CLI") {
			t.Fatalf("err = %v, want manual CLI refusal", err)
		}
	}
}

func TestDeleteRequiresTerminal(t *testing.T) {
	t.Setenv("OBK_AGENT", "")
	err := runDelete(chatsDeleteCmd, true)
	if err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("err = %v, want terminal refusal", err)
	}
}
