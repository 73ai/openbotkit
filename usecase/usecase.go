package usecase

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/internal/testutil"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/service/scheduler"
	"github.com/73ai/openbotkit/source/websearch"
	"github.com/73ai/openbotkit/spectest"

	_ "github.com/73ai/openbotkit/provider/anthropic"
	_ "github.com/73ai/openbotkit/provider/cerebras"
	_ "github.com/73ai/openbotkit/provider/gemini"
	_ "github.com/73ai/openbotkit/provider/groq"
	_ "github.com/73ai/openbotkit/provider/openai"
	_ "github.com/73ai/openbotkit/provider/openrouter"
	_ "github.com/73ai/openbotkit/provider/zai"
)

// DefaultProfile is the profile used by use case tests.
var DefaultProfile = config.Profiles["gemini"]

// Fixture wraps spectest.LocalFixture with profile-based provider setup.
type Fixture struct {
	*spectest.LocalFixture
	ProfileName  string
	mainProvider provider.Provider
	mainModel    string
}

// NewFixture creates a use case test fixture with profile-based providers.
func NewFixture(t *testing.T) *Fixture {
	t.Helper()

	if os.Getenv("OBK_USECASE") == "" {
		t.Skip("use case tests require OBK_USECASE=1")
	}

	testutil.LoadEnv(t)

	for _, name := range DefaultProfile.Providers {
		envVar := provider.ProviderEnvVars[name]
		if envVar == "" {
			continue
		}
		if os.Getenv(envVar) == "" {
			t.Skipf("skipping: missing API key for %s (need %s)", name, envVar)
		}
	}

	env := spectest.NewEnv(t)

	models := &config.ModelsConfig{
		Default: DefaultProfile.Tiers.Default,
		Complex: DefaultProfile.Tiers.Complex,
		Fast:    DefaultProfile.Tiers.Fast,
		Nano:    DefaultProfile.Tiers.Nano,
	}

	reg, err := provider.NewRegistry(models)
	if err != nil {
		t.Fatalf("create provider registry: %v", err)
	}

	mainProvider, mainModel, err := resolveSpec(reg, models.Default)
	if err != nil {
		t.Fatalf("resolve default model: %v", err)
	}

	judgeProvider, judgeModel, err := resolveSpec(reg, models.Complex)
	if err != nil {
		t.Fatalf("resolve complex/judge model: %v", err)
	}

	env.JudgeProvider = judgeProvider
	env.JudgeModel = judgeModel

	return &Fixture{
		LocalFixture: env,
		ProfileName:  DefaultProfile.Name,
		mainProvider: mainProvider,
		mainModel:    mainModel,
	}
}

func resolveSpec(reg *provider.Registry, spec string) (provider.Provider, string, error) {
	name, model, err := provider.ParseModelSpec(spec)
	if err != nil {
		return nil, "", err
	}
	p, ok := reg.Get(name)
	if !ok {
		return nil, "", fmt.Errorf("provider %q not in registry", name)
	}
	return p, model, nil
}

// SchedDBPath returns the path to the scheduler database.
func (f *Fixture) SchedDBPath() string {
	return filepath.Join(f.Dir(), "scheduler", "data.db")
}

func (f *Fixture) webDeps() tools.WebToolDeps {
	ws := websearch.New(websearch.Config{})
	return tools.WebToolDeps{
		WS:       ws,
		Provider: f.mainProvider,
		Model:    f.mainModel,
	}
}

// Agent creates an agent using the profile's default provider and model.
func (f *Fixture) Agent(t *testing.T) *agent.Agent {
	t.Helper()

	webDeps := f.webDeps()

	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewBashTool(30 * time.Second))
	toolReg.Register(&tools.FileReadTool{})
	toolReg.Register(&tools.LoadSkillsTool{})
	toolReg.Register(&tools.SearchSkillsTool{})
	toolReg.Register(tools.NewWebSearchTool(webDeps))
	toolReg.Register(tools.NewWebFetchTool(webDeps))

	identity := "You are a personal AI assistant powered by OpenBotKit.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg)

	return agent.New(f.mainProvider, f.mainModel, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(15),
	)
}

// ScheduleAgent creates an agent with schedule tools registered.
func (f *Fixture) ScheduleAgent(t *testing.T) *agent.Agent {
	t.Helper()

	schedDeps := tools.ScheduleToolDeps{
		Cfg: &config.Config{
			Scheduler: &config.SchedulerConfig{
				Storage: config.StorageConfig{Driver: "sqlite", DSN: f.SchedDBPath()},
			},
		},
		Channel: "telegram",
		ChannelMeta: scheduler.ChannelMeta{
			BotToken: "test-token",
			OwnerID:  42,
		},
	}

	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewBashTool(30 * time.Second))
	toolReg.Register(&tools.FileReadTool{})
	toolReg.Register(&tools.LoadSkillsTool{})
	toolReg.Register(&tools.SearchSkillsTool{})
	toolReg.Register(tools.NewCreateScheduleTool(schedDeps))
	toolReg.Register(tools.NewListSchedulesTool(schedDeps))
	toolReg.Register(tools.NewDeleteScheduleTool(schedDeps))

	identity := "You are a personal AI assistant powered by OpenBotKit.\n"
	extras := "\nThe user's timezone is America/New_York.\nToday's date is " + time.Now().Format("2006-01-02") + ".\n" +
		"You are connected to the user via Telegram. Scheduled tasks will be delivered on Telegram automatically.\n" +
		"When the user asks for something to happen regularly (every day, every morning, weekly, etc.) or at a specific future time, use the create_schedule tool.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg, extras)

	return agent.New(f.mainProvider, f.mainModel, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(15),
	)
}
