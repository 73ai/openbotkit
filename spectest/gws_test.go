package spectest

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestSpec_ListRecentGoogleDocs verifies the agent loads the correct skill
// (gws-drive, not gws-docs) and uses the right command (drive files list)
// when asked to list Google Docs.
func TestSpec_ListRecentGoogleDocs(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.InstallGWSSkills(t)

		runner := &GWSMockRunner{
			Responses: map[string]string{
				"drive files list": `{"files":[{"id":"doc1","name":"Project Roadmap Q3","mimeType":"application/vnd.google-apps.document","modifiedTime":"2025-01-15T12:00:00Z"},{"id":"doc2","name":"Meeting Notes Jan 14","mimeType":"application/vnd.google-apps.document","modifiedTime":"2025-01-14T10:00:00Z"},{"id":"doc3","name":"Design Spec v2","mimeType":"application/vnd.google-apps.document","modifiedTime":"2025-01-13T08:30:00Z"}]}`,
			},
		}

		a := fx.GWSAgent(t, runner)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "Check my recent Google Docs"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)

		// Verify the agent used drive files list, not docs list.
		runner.mu.Lock()
		calls := runner.Calls
		runner.mu.Unlock()

		if len(calls) == 0 {
			t.Fatal("agent never called gws_execute")
		}

		var usedDriveFilesList bool
		var usedDocsListWrong bool
		for _, c := range calls {
			joined := strings.Join(c.Args, " ")
			normalized := strings.ReplaceAll(joined, ".", " ")
			if strings.HasPrefix(normalized, "drive files list") {
				usedDriveFilesList = true
			}
			if strings.HasPrefix(normalized, "docs list") || strings.HasPrefix(normalized, "docs documents list") {
				usedDocsListWrong = true
			}
		}

		if usedDocsListWrong && !usedDriveFilesList {
			t.Error("agent used wrong command: 'docs list' instead of 'drive files list' — listing docs requires the Drive API")
		}
		if !usedDriveFilesList {
			var cmds []string
			for _, c := range calls {
				cmds = append(cmds, strings.Join(c.Args, " "))
			}
			t.Errorf("agent did not use 'drive files list'; commands used: %v", cmds)
		}

		fx.AssertJudge(t, prompt, result,
			"The response should mention the user's Google Docs. It should reference at least some of: "+
				"'Project Roadmap Q3', 'Meeting Notes Jan 14', or 'Design Spec v2'.")
	})
}

// TestSpec_CheckCalendarEvents verifies the agent loads gws-calendar skill
// and uses the correct command to list events.
func TestSpec_CheckCalendarEvents(t *testing.T) {
	EachProvider(t, func(t *testing.T, fx *LocalFixture) {
		fx.InstallGWSSkills(t)

		runner := &GWSMockRunner{
			Responses: map[string]string{
				"calendar events list": `{"items":[{"summary":"Team Standup","start":{"dateTime":"2025-01-15T09:00:00Z"},"end":{"dateTime":"2025-01-15T09:30:00Z"}},{"summary":"Lunch with Sarah","start":{"dateTime":"2025-01-15T12:00:00Z"},"end":{"dateTime":"2025-01-15T13:00:00Z"}}]}`,
			},
		}

		a := fx.GWSAgent(t, runner)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		prompt := "What's on my calendar today?"
		result, err := a.Run(ctx, prompt)
		if err != nil {
			t.Fatalf("agent.Run: %v", err)
		}

		AssertNotEmpty(t, result)

		runner.mu.Lock()
		calls := runner.Calls
		runner.mu.Unlock()

		if len(calls) == 0 {
			t.Fatal("agent never called gws_execute")
		}

		var usedCalendarEventsList bool
		for _, c := range calls {
			joined := strings.Join(c.Args, " ")
			normalized := strings.ReplaceAll(joined, ".", " ")
			if strings.HasPrefix(normalized, "calendar events list") {
				usedCalendarEventsList = true
			}
		}
		if !usedCalendarEventsList {
			var cmds []string
			for _, c := range calls {
				cmds = append(cmds, strings.Join(c.Args, " "))
			}
			t.Errorf("agent did not use 'calendar events list'; commands used: %v", cmds)
		}

		fx.AssertJudge(t, prompt, result,
			"The response should mention calendar events. It should reference 'Team Standup' and/or 'Lunch with Sarah'.")
	})
}
