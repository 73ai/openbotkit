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
	"github.com/73ai/openbotkit/service/learnings"
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

// defaultProfileName is the profile used by use case tests.
const defaultProfileName = "gemini"

// Fixture wraps spectest.LocalFixture with profile-based provider setup.
type Fixture struct {
	*spectest.LocalFixture
	mainProvider provider.Provider
	mainModel    string
	fastProvider provider.Provider
	fastModel    string
}

// NewFixture creates a use case test fixture with profile-based providers.
func NewFixture(t *testing.T) *Fixture {
	t.Helper()

	if os.Getenv("OBK_USECASE") == "" {
		t.Skip("use case tests require OBK_USECASE=1")
	}

	testutil.LoadEnv(t)

	profile, ok := config.Profiles[defaultProfileName]
	if !ok {
		t.Fatalf("profile %q not found in config.Profiles", defaultProfileName)
	}

	for _, name := range profile.Providers {
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
		Default: profile.Tiers.Default,
		Complex: profile.Tiers.Complex,
		Fast:    profile.Tiers.Fast,
		Nano:    profile.Tiers.Nano,
	}

	reg, err := provider.NewRegistry(models)
	if err != nil {
		t.Fatalf("create provider registry: %v", err)
	}

	mainProvider, mainModel, err := resolveSpec(reg, models.Default)
	if err != nil {
		t.Fatalf("resolve default model: %v", err)
	}

	fastProvider, fastModel := tools.ResolveFastProvider(models, reg, mainProvider, mainModel)

	judgeProvider, judgeModel, err := resolveSpec(reg, models.Complex)
	if err != nil {
		t.Fatalf("resolve complex/judge model: %v", err)
	}

	env.JudgeProvider = judgeProvider
	env.JudgeModel = judgeModel

	return &Fixture{
		LocalFixture: env,
		mainProvider: mainProvider,
		mainModel:    mainModel,
		fastProvider: fastProvider,
		fastModel:    fastModel,
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

// testInteractor auto-approves all actions for use case tests.
type testInteractor struct{}

func (t *testInteractor) Notify(_ string) error                  { return nil }
func (t *testInteractor) NotifyLink(_, _ string) error           { return nil }
func (t *testInteractor) RequestApproval(_ string) (bool, error) { return true, nil }

// SchedDBPath returns the path to the scheduler database.
func (f *Fixture) SchedDBPath() string {
	return filepath.Join(f.Dir(), "scheduler", "data.db")
}

// WorkspaceDir returns the workspace directory for persistent research artifacts.
func (f *Fixture) WorkspaceDir() string {
	return filepath.Join(f.Dir(), "workspace")
}

// Agent creates an agent that mirrors the production Telegram agent setup:
// standard tools + schedule + web + learnings + subagent.
func (f *Fixture) Agent(t *testing.T) *agent.Agent {
	t.Helper()

	// Standard tools: bash, file_read, file_write, file_edit,
	// load_skills, search_skills, dir_explore, content_search, sandbox_exec
	toolReg := tools.NewStandardRegistry(nil, nil)

	// Schedule tools
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
	toolReg.Register(tools.NewCreateScheduleTool(schedDeps))
	toolReg.Register(tools.NewListSchedulesTool(schedDeps))
	toolReg.Register(tools.NewDeleteScheduleTool(schedDeps))

	// Web tools
	ws := websearch.New(websearch.Config{})
	webDeps := tools.WebToolDeps{
		WS:       ws,
		Provider: f.fastProvider,
		Model:    f.fastModel,
	}
	toolReg.Register(tools.NewWebSearchTool(webDeps))
	toolReg.Register(tools.NewWebFetchTool(webDeps))

	// Learnings tools
	learningsDir := filepath.Join(f.Dir(), "learnings")
	if err := os.MkdirAll(learningsDir, 0700); err != nil {
		t.Fatalf("mkdir learnings: %v", err)
	}
	learningsDeps := tools.LearningsDeps{Store: learnings.New(learningsDir)}
	toolReg.Register(tools.NewLearningSaveTool(learningsDeps))
	toolReg.Register(tools.NewLearningReadTool(learningsDeps))
	toolReg.Register(tools.NewLearningSearchTool(learningsDeps))

	// Workspace
	workspaceDir := f.WorkspaceDir()
	if err := os.MkdirAll(workspaceDir, 0700); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	// Subagent with enriched tools
	subagentDeps := tools.SubagentRegistryDeps{
		WebDeps:       &webDeps,
		LearningsDeps: &learningsDeps,
		ScheduleDeps:  &schedDeps,
	}
	toolReg.Register(tools.NewSubagentTool(tools.SubagentConfig{
		Provider: f.mainProvider,
		Model:    f.mainModel,
		ToolFactory: func() *tools.Registry {
			return tools.NewSubagentRegistry(subagentDeps)
		},
		System: "You are a focused sub-agent. Complete the given task and return a concise result.",
		Extras: []string{"\nWorkspace directory: " + workspaceDir + "\n"},
	}))

	identity := "You are a personal AI assistant communicating via Telegram.\n"
	extras := "\nThe user's timezone is America/New_York.\nToday's date is " + time.Now().Format("2006-01-02") + ".\n" +
		"\nWorkspace directory: " + workspaceDir + "\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg, extras)

	return agent.New(f.mainProvider, f.mainModel, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(15),
	)
}

// AgentWithDelegation creates an agent with delegate_task and check_task
// tools registered. Returns nil if no external agents are detected.
func (f *Fixture) AgentWithDelegation(t *testing.T, agents []tools.AgentInfo) *agent.Agent {
	t.Helper()

	toolReg := tools.NewStandardRegistry(nil, nil)

	ws := websearch.New(websearch.Config{})
	webDeps := tools.WebToolDeps{
		WS:       ws,
		Provider: f.fastProvider,
		Model:    f.fastModel,
	}
	toolReg.Register(tools.NewWebSearchTool(webDeps))
	toolReg.Register(tools.NewWebFetchTool(webDeps))

	learningsDir := filepath.Join(f.Dir(), "learnings")
	os.MkdirAll(learningsDir, 0700)
	learningsDeps := tools.LearningsDeps{Store: learnings.New(learningsDir)}
	toolReg.Register(tools.NewLearningSaveTool(learningsDeps))
	toolReg.Register(tools.NewLearningReadTool(learningsDeps))
	toolReg.Register(tools.NewLearningSearchTool(learningsDeps))

	workspaceDir := f.WorkspaceDir()
	os.MkdirAll(workspaceDir, 0700)

	scratchDir := filepath.Join(f.Dir(), "scratch")
	os.MkdirAll(scratchDir, 0700)

	inter := &testInteractor{}
	tracker := tools.NewTaskTracker()
	toolReg.Register(tools.NewDelegateTaskTool(tools.DelegateTaskConfig{
		Interactor: inter,
		Agents:     agents,
		Tracker:    tracker,
		ScratchDir: scratchDir,
	}))
	toolReg.Register(tools.NewCheckTaskTool(tracker))

	identity := "You are a personal AI assistant communicating via Telegram.\n"
	extras := "\nThe user's timezone is America/New_York.\nToday's date is " + time.Now().Format("2006-01-02") + ".\n" +
		"\nWorkspace directory: " + workspaceDir + "\n" +
		"\nAlways deliver results directly. Do not ask the user if they want to see results — just present them.\n"
	blocks := tools.BuildSystemBlocks(identity, toolReg, extras)

	return agent.New(f.mainProvider, f.mainModel, toolReg,
		agent.WithSystemBlocks(blocks),
		agent.WithMaxIterations(15),
	)
}
