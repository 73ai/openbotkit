package scheduler

import (
	"encoding/json"
	"testing"
)

func TestChannelMetaJSONRoundTrip(t *testing.T) {
	orig := ChannelMeta{
		BotToken:  "tok123",
		OwnerID:   42,
		Workspace: "myteam",
		ChannelID: "C999",
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ChannelMeta
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != orig {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, orig)
	}
}
