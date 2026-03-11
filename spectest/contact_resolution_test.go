package spectest

import (
	"context"
	"testing"
	"time"
)

func TestSpec_ResolveContactAndSendWhatsApp(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenContacts(t, []ContactFixture{
			{
				Name:        "David Chen",
				Phones:      []string{"+919876543210"},
				WhatsAppJID: "919876543210@s.whatsapp.net",
				Aliases:     []string{"David", "Dave"},
			},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Tell David I'll be late"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The agent must have attempted to send a WhatsApp message to David. "+
				"The response may mention David Chen, a WhatsApp JID, or simply indicate "+
				"that a message was sent or that sending failed due to an auth/connection error. "+
				"Any of these outcomes count as a PASS. The only failure is if the agent "+
				"did not attempt to contact David at all or said it cannot find David.")
	})
}

func TestSpec_ResolveContactAmbiguous(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenContacts(t, []ContactFixture{
			{
				Name:        "David Chen",
				Phones:      []string{"+919876543210"},
				WhatsAppJID: "919876543210@s.whatsapp.net",
				Aliases:     []string{"David"},
			},
			{
				Name:        "David Miller",
				Phones:      []string{"+14155551234"},
				WhatsAppJID: "14155551234@s.whatsapp.net",
				Aliases:     []string{"David"},
			},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Tell David I'll be late"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The agent found multiple contacts named David (David Chen and David Miller). "+
				"It must have indicated the ambiguity — either by asking the user to clarify "+
				"which David, or by listing both matches. The agent should NOT have silently "+
				"picked one David without acknowledging the ambiguity.")
	})
}

func TestSpec_ResolveContactForEmail(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenContacts(t, []ContactFixture{
			{
				Name:    "Alice Smith",
				Emails:  []string{"alice@acme.com"},
				Aliases: []string{"Alice"},
			},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Email Alice about the meeting tomorrow"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		fx.AssertJudge(t, prompt, result,
			"The agent must have attempted to send or draft an email to Alice. "+
				"The response may mention Alice Smith, alice@acme.com, or simply indicate "+
				"that an email was sent or that sending failed due to an auth/config error. "+
				"Any of these outcomes count as a PASS. The only failure is if the agent "+
				"did not attempt to email Alice at all or said it cannot find Alice.")
	})
}
