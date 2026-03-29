package usecase

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/73ai/openbotkit/agent"
	"github.com/73ai/openbotkit/agent/tools"
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/internal/testutil"
	"github.com/73ai/openbotkit/provider"
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

// Agent creates an agent using the profile's default provider and model.
func (f *Fixture) Agent(t *testing.T) *agent.Agent {
	t.Helper()

	toolReg := tools.NewRegistry()
	toolReg.Register(tools.NewBashTool(30 * time.Second))
	toolReg.Register(&tools.FileReadTool{})
	toolReg.Register(&tools.LoadSkillsTool{})
	toolReg.Register(&tools.SearchSkillsTool{})

	identity := "You are a personal AI assistant powered by OpenBotKit.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg)

	return agent.New(f.mainProvider, f.mainModel, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(15),
	)
}
