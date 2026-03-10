package spectest

import (
	"context"
	"testing"
	"time"
)

// TestSpec_SummarizeCommunicationsAcrossSources seeds emails and WhatsApp
// messages from the same person, then asks the agent to summarize all
// communications. The agent must autonomously discover and use both email-read
// and whatsapp-read skills, query both databases, and synthesize the results
// in a single turn.
func TestSpec_SummarizeCommunicationsAcrossSources(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenEmails(t, []Email{
			{From: "alice@acme.com", To: "me@example.com", Subject: "Q3 Budget Review", Body: "Hi, please review the Q3 budget spreadsheet I shared. We need to finalize numbers by Friday."},
			{From: "alice@acme.com", To: "me@example.com", Subject: "Team Offsite in Portland", Body: "I'm thinking we do the offsite in Portland in October. Thoughts?"},
		})

		fx.GivenWhatsAppMessages(t, []WhatsAppMessage{
			{SenderJID: "alice@s.whatsapp.net", SenderName: "Alice", ChatJID: "alice@s.whatsapp.net", ChatName: "Alice", Text: "Booked Trattoria Vecchia for Friday dinner, confirmation code TRV-8842."},
			{SenderJID: "alice@s.whatsapp.net", SenderName: "Alice", ChatJID: "alice@s.whatsapp.net", ChatName: "Alice", Text: "Can you bring the Nakamura prototype to the offsite? Serial number NK-2047."},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Summarize all communications from Alice across both email and WhatsApp."
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		AssertJudge(t, fx.Provider, fx.Model, prompt, result,
			"The response must cover topics from BOTH email and WhatsApp. "+
				"It should mention the Q3 budget review or Portland offsite from email, AND reference "+
				"Trattoria Vecchia, TRV-8842, Nakamura prototype, or NK-2047 from WhatsApp. "+
				"It should not claim that data from one source is missing if it was provided.")
	})
}

// TestSpec_RecallMemoryAndCorrelateEmails seeds personal memories about a
// relationship and emails with project details. The agent must autonomously
// check memories for context about the person, search emails for specifics,
// and combine both into a coherent answer in a single turn.
func TestSpec_RecallMemoryAndCorrelateEmails(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.GivenMemories(t, []UserMemory{
			{Content: "Raj Patel is my tech lead at Zephyr Industries", Category: "relationship"},
			{Content: "Project Firebird has a hard deadline of June 15, 2025", Category: "project"},
		})

		fx.GivenEmails(t, []Email{
			{From: "raj.patel@zephyr.io", To: "me@example.com", Subject: "Project Firebird Sprint 7 Retro", Body: "Sprint 7 retro is scheduled for May 22. Please prepare your notes on the auth module refactor."},
			{From: "raj.patel@zephyr.io", To: "me@example.com", Subject: "Project Firebird Launch Prep", Body: "Client confirmed the staging demo for June 10. We need all QA passed by June 8."},
		})

		a := fx.Agent(t)
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		prompt := "Tell me everything about Raj and Project Firebird. Check both my memories and emails."
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)
		AssertJudge(t, fx.Provider, fx.Model, prompt, result,
			"The response must include information from BOTH memories and emails. "+
				"It should mention Raj Patel is the tech lead at Zephyr Industries (from memory) AND mention "+
				"email subjects or content about Project Firebird Sprint 7 and Launch Prep (from emails). "+
				"It should not only use one source.")
	})
}
