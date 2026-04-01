package self_improvement

import (
	"context"
	"testing"
	"time"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/usecase"
)

func TestUseCase_ModifyConfig(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Change the timezone to America/Los_Angeles using obk config set")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Timezone != "America/Los_Angeles" {
		t.Errorf("expected timezone America/Los_Angeles, got %q", cfg.Timezone)
	}
}

func TestUseCase_ModifyWorkspace(t *testing.T) {
	fx := usecase.NewFixture(t)
	a := fx.Agent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Set the workspace directory to /tmp/my-workspace using obk config set")
	if err != nil {
		t.Fatalf("agent run: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Workspace != "/tmp/my-workspace" {
		t.Errorf("expected workspace /tmp/my-workspace, got %q", cfg.Workspace)
	}
}
