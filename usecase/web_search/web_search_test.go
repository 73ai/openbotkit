package web_search

import (
	"context"
	"testing"
	"time"

	"github.com/73ai/openbotkit/spectest"
	"github.com/73ai/openbotkit/usecase"
)

func TestUseCase_SearchSynthesizeEvent(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	prompt := "Hey what happened with the CrowdStrike outage? I keep hearing about it but missed the details"
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, prompt, result,
		"Response should explain the CrowdStrike incident — a faulty software update that caused widespread Windows system crashes/outages. Must read as a coherent summary, not a list of search result titles.")
}

func TestUseCase_SearchStoreHours(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	prompt := "Is Costco open right now? I'm near the one in Foster City, CA"
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, prompt, result,
		"Response should address whether Costco is currently open, referencing store hours or days of operation. Must not say it cannot determine hours.")
}

func TestUseCase_SearchVersionCheck(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	prompt := "Can you check if there's a new version of Postgres out? I'm on 16.2 and wondering if I should upgrade"
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, prompt, result,
		"Response should mention a PostgreSQL version newer than 16.2 and acknowledge the user is on 16.2. Must not say it cannot find version information.")
}

func TestUseCase_SearchStockPrice(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	prompt := "What's AAPL trading at right now?"
	result, err := a.Run(ctx, prompt)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	spectest.AssertNotEmpty(t, result)
	fx.AssertJudge(t, prompt, result,
		"Response should mention a dollar price for AAPL/Apple stock. Must include a numeric value, not just say check a financial website.")
}
