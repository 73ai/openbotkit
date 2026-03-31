package x_read

import (
	"context"
	"testing"
	"time"

	"github.com/73ai/openbotkit/source/twitter/client"
	"github.com/73ai/openbotkit/spectest"
	"github.com/73ai/openbotkit/usecase"
)

func skipUnlessXConnected(t *testing.T) {
	t.Helper()
	session, err := client.LoadSession()
	if err != nil {
		t.Skip("X not connected: run 'obk x auth login' first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.ValidateSession(ctx, session); err != nil {
		t.Skipf("X session expired or invalid: %v", err)
	}
}

func TestUseCase_XReadTimeline(t *testing.T) {
	fx := usecase.NewFixture(t)
	skipUnlessXConnected(t)

	xRead := fx.LoadSkillContent(t, "x-read")
	a := fx.Agent(t, xRead)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	result, err := a.Run(ctx, "Sync my X timeline and show me the 5 most recent posts")
	if err != nil {
		t.Fatalf("read timeline: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, "Sync my X timeline and show me the 5 most recent posts", result,
		"The response should contain actual posts from X/Twitter, including usernames or post content. It should not say it cannot access X.")
}

func TestUseCase_XSearchPosts(t *testing.T) {
	fx := usecase.NewFixture(t)
	skipUnlessXConnected(t)

	xRead := fx.LoadSkillContent(t, "x-read")
	a := fx.Agent(t, xRead)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Turn 1: sync timeline to populate the local DB
	_, err := a.Run(ctx, "Sync my X timeline")
	if err != nil {
		t.Fatalf("turn 1 (sync): %v", err)
	}

	// Turn 2: search local posts
	result, err := a.Run(ctx, `Search my local X posts for "AI" using the --local flag`)
	if err != nil {
		t.Fatalf("turn 2 (search): %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, `Search my local X posts for "AI"`, result,
		"The response should contain posts from X/Twitter related to AI, including post content or usernames.")
}
