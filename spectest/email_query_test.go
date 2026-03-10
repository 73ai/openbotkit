package spectest

import (
	"context"
	"testing"
	"time"
)

func TestSpec_FindEmailsBySender(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenEmails(t, []Email{
			{From: "alice@example.com", Subject: "Meeting Tomorrow", Body: "Let's meet at 2pm"},
			{From: "bob@example.com", Subject: "Project Update", Body: "Here is the latest"},
			{From: "alice@example.com", Subject: "Lunch Plans", Body: "Friday lunch?"},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "Find emails from Alice"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		AssertJudge(t, fx.Provider, fx.Model, prompt, result,
			"The response must list Alice's emails. It should mention both 'Meeting Tomorrow' and 'Lunch Plans' subjects. "+
				"It should NOT include Bob's 'Project Update' email.")
	})
}
