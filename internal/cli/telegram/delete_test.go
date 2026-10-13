package telegram

import (
	"strings"
	"testing"
)

func TestDeleteRejectsAgents(t *testing.T) {
	t.Setenv("OBK_AGENT", "1")
	for _, err := range []error{runChatDelete(chatsDeleteCmd, nil), runMessagesDelete(messagesDeleteCmd, nil)} {
		if err == nil || !strings.Contains(err.Error(), "manual CLI") {
			t.Fatalf("err = %v, want manual CLI refusal", err)
		}
	}
}
