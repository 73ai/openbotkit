package spectest

import (
	"context"
	"testing"
	"time"
)

func TestSpec_FindEmailsBySender(t *testing.T) {
	fx := NewLocalFixture(t)

	fx.GivenEmails(t, []Email{
		{From: "alice@example.com", Subject: "Meeting Tomorrow", Body: "Let's meet at 2pm"},
		{From: "bob@example.com", Subject: "Project Update", Body: "Here is the latest"},
		{From: "alice@example.com", Subject: "Lunch Plans", Body: "Friday lunch?"},
	})

	a := fx.Agent(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := a.Run(ctx, "Find emails from Alice")
	if err != nil {
		t.Fatalf("agent.Run: %v", err)
	}

	AssertNotEmpty(t, result)
	AssertContains(t, result, "alice", "Meeting Tomorrow")
}
