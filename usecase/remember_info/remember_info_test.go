package remember_info

import (
	"context"
	"testing"
	"time"

	"github.com/73ai/openbotkit/spectest"
	"github.com/73ai/openbotkit/usecase"
)

func TestUseCase_RememberDoorCode(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Turn 1: tell the bot a door code
	_, err := a.Run(ctx, "Remember this: my front door code is 4521")
	if err != nil {
		t.Fatalf("turn 1 (save): %v", err)
	}

	// Turn 2: ask for it back
	result, err := a.Run(ctx, "What's my door code?")
	if err != nil {
		t.Fatalf("turn 2 (recall): %v", err)
	}

	spectest.AssertContains(t, result, "4521")
	fx.AssertJudge(t, "What's my door code?", result,
		"The agent should recall that the door code is 4521.")
}

func TestUseCase_RecallWithoutSave(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	result, err := a.Run(ctx, "What's my door code?")
	if err != nil {
		t.Fatalf("recall without save: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, "What's my door code?", result,
		"The agent should indicate it does not know or has no record of a door code. It must NOT fabricate a code.")
}
