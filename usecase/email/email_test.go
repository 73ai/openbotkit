package email

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/73ai/openbotkit/spectest"
	"github.com/73ai/openbotkit/usecase"

	_ "github.com/73ai/openbotkit/provider/gemini"
)

func TestUseCase_EmailReadSummarize(t *testing.T) {
	fx := usecase.NewFixture(t)

	fx.GivenEmails(t, []spectest.Email{
		{From: "ops@company.com", Subject: "URGENT: Server outage in prod", Body: "Production database is down. All services returning 500 errors. Need immediate attention."},
		{From: "newsletter@techdigest.com", Subject: "Weekly Tech Digest #42", Body: "Top stories this week: AI advances, new framework releases, cloud pricing updates."},
		{From: "alice@company.com", Subject: "Budget meeting tomorrow at 2pm", Body: "Hi, please review the Q2 budget spreadsheet before our meeting tomorrow. I need your sign-off."},
		{From: "promo@store.com", Subject: "50% OFF EVERYTHING!", Body: "Limited time offer! Shop now and save big on all items."},
		{From: "cto@company.com", Subject: "Re: Architecture review needed", Body: "Can you review the new microservices proposal by end of week? Blocking the next sprint planning."},
	})

	emailSkill := fx.LoadSkillContent(t, "email-read")
	a := fx.Agent(t, emailSkill)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	result, err := a.Run(ctx, "What emails need my attention?")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertChecklist(t, result, []string{
		"Does the response mention the server outage email from ops@company.com?",
		"Does the response mention the budget meeting from alice@company.com?",
		"Does the response mention the architecture review from cto@company.com?",
		"Does the response NOT treat the newsletter or promo emails as needing attention?",
	})
}

func TestUseCase_EmailSendWithApproval(t *testing.T) {
	fx := usecase.NewFixture(t)

	fx.GivenEmails(t, []spectest.Email{
		{From: "alice@company.com", Subject: "Budget meeting tomorrow at 2pm", Body: "Hi, please review the Q2 budget spreadsheet before our meeting. I need your sign-off by EOD."},
	})
	fx.GivenContacts(t, []spectest.ContactFixture{
		{Name: "Alice", Emails: []string{"alice@company.com"}},
	})

	dryRunPath := filepath.Join(fx.Dir(), "dry-run-send.json")
	t.Setenv("OBK_GMAIL_DRY_RUN", dryRunPath)

	inter := &usecase.RecordingInteractor{}
	emailReadSkill := fx.LoadSkillContent(t, "email-read")
	emailSendSkill := fx.LoadSkillContent(t, "email-send")
	a := fx.AgentWithInteractor(t, inter, emailReadSkill, emailSendSkill)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Reply to Alice about the budget meeting — tell her I'll have the spreadsheet reviewed by noon tomorrow.")
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Verify approval was requested
	if len(inter.Approvals) == 0 {
		t.Error("expected at least one approval request for email send")
	} else {
		found := false
		for _, desc := range inter.Approvals {
			if strings.Contains(desc, "obk gmail send") || strings.Contains(desc, "obk gmail drafts") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected approval for email send command, got: %v", inter.Approvals)
		}
	}

	// Verify dry-run file
	data, err := os.ReadFile(dryRunPath)
	if err != nil {
		t.Fatalf("read dry-run file: %v", err)
	}
	var sent map[string]any
	if err := json.Unmarshal(data, &sent); err != nil {
		t.Fatalf("parse dry-run json: %v", err)
	}

	// Check recipient
	to, ok := sent["to"].([]any)
	if !ok || len(to) == 0 {
		t.Fatal("dry-run: missing 'to' field")
	}
	toStr, _ := to[0].(string)
	if !strings.Contains(strings.ToLower(toStr), "alice") {
		t.Errorf("dry-run: expected recipient to contain alice, got %q", toStr)
	}

	// Check subject exists
	subject, _ := sent["subject"].(string)
	if subject == "" {
		t.Error("dry-run: subject is empty")
	}

	// Check body mentions budget/spreadsheet/review/noon
	body, _ := sent["body"].(string)
	if body == "" {
		t.Fatal("dry-run: body is empty")
	}
	bodyLower := strings.ToLower(body)
	if !strings.Contains(bodyLower, "budget") && !strings.Contains(bodyLower, "spreadsheet") && !strings.Contains(bodyLower, "review") && !strings.Contains(bodyLower, "noon") {
		t.Errorf("dry-run: body should mention budget/spreadsheet/review/noon, got: %s", body)
	}
}

func TestUseCase_UrgentEmailTelegramAlert(t *testing.T) {
	fx := usecase.NewFixture(t)

	// Seed emails directly via GivenEmails (returns them in the DB)
	fx.GivenEmails(t, []spectest.Email{
		{From: "ops@company.com", Subject: "URGENT: Server outage - all systems down", Body: "Production is completely down. HTTP 500 on all endpoints. Customer-facing services are unreachable. Immediate response required."},
		{From: "newsletter@techdigest.com", Subject: "Your Weekly Tech Newsletter", Body: "This week in tech: new AI models released, React 20 announced."},
		{From: "hr@company.com", Subject: "Reminder: Submit your timesheet", Body: "Please submit your timesheet for this pay period by Friday."},
	})

	// Get email IDs from the DB
	emailIDs := fx.EmailIDs(t)
	if len(emailIDs) != 3 {
		t.Fatalf("expected 3 emails, got %d", len(emailIDs))
	}

	// Set up hook + channel registry + River to process the hook
	pusher := &spectest.CapturePusher{}
	fx.RunEventHook(t, pusher, emailIDs, fx.Models)

	// Verify: should have pushed a notification about the server outage
	messages := pusher.Messages()
	if len(messages) == 0 {
		t.Fatal("expected at least one push notification for urgent email")
	}

	found := false
	for _, msg := range messages {
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "outage") || strings.Contains(lower, "server") || strings.Contains(lower, "down") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected push notification to mention server outage, got: %v", messages)
	}

	// Should NOT have pushed anything about the newsletter or timesheet
	for _, msg := range messages {
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "newsletter") || strings.Contains(lower, "timesheet") {
			t.Errorf("unexpected push for routine email: %s", msg)
		}
	}
}
