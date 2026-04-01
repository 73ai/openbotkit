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
	configSkill := fx.LoadSkillContent(t, "config-manage")
	a := fx.Agent(t, configSkill)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Change my timezone to America/Los_Angeles")
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
	configSkill := fx.LoadSkillContent(t, "config-manage")
	a := fx.Agent(t, configSkill)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	_, err := a.Run(ctx, "Set my workspace directory to /tmp/my-workspace")
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
