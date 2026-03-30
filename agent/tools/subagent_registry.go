package tools

import (
	"errors"

	"github.com/73ai/openbotkit/source/slack"
)

// denyInteractor implements Interactor for sub-agent contexts where
// user interaction is not available. Matches Claude Code's background
// subagent model: guarded tools fail, subagent continues.
type denyInteractor struct{}

func (d *denyInteractor) Notify(_ string) error                { return nil }
func (d *denyInteractor) NotifyLink(_, _ string) error         { return nil }
func (d *denyInteractor) RequestApproval(_ string) (bool, error) {
	return false, errors.New("not permitted in sub-agent context")
}

// SubagentRegistryDeps provides optional dependencies for building
// an enriched subagent tool registry.
type SubagentRegistryDeps struct {
	ScratchDir    string
	WebDeps       *WebToolDeps
	LearningsDeps *LearningsDeps
	ScheduleDeps  *ScheduleToolDeps
	SlackClient   slack.API
	Agents        []AgentInfo
}

// NewSubagentRegistry creates a tool registry for subagent contexts.
// All tools are registered so the subagent sees them. Safe tools work
// normally; guarded tools get a denyInteractor so they fail via the
// existing GuardedAction error path. The subagent tool itself is
// excluded to prevent recursion.
func NewSubagentRegistry(deps SubagentRegistryDeps) *Registry {
	deny := &denyInteractor{}

	r := NewStandardRegistry(nil, nil)
	if deps.ScratchDir != "" {
		r.SetScratchDir(deps.ScratchDir)
	}

	// Safe tools — no guard, always work.
	if deps.WebDeps != nil {
		r.Register(NewWebSearchTool(*deps.WebDeps))
		r.Register(NewWebFetchTool(*deps.WebDeps))
	}
	if deps.LearningsDeps != nil {
		r.Register(NewLearningSaveTool(*deps.LearningsDeps))
		r.Register(NewLearningReadTool(*deps.LearningsDeps))
		r.Register(NewLearningSearchTool(*deps.LearningsDeps))
	}
	if deps.ScheduleDeps != nil {
		r.Register(NewCreateScheduleTool(*deps.ScheduleDeps))
		r.Register(NewListSchedulesTool(*deps.ScheduleDeps))
		r.Register(NewDeleteScheduleTool(*deps.ScheduleDeps))
	}

	// Guarded tools — denyInteractor makes them fail via GuardedAction.
	if deps.SlackClient != nil {
		readDeps := SlackToolDeps{Client: deps.SlackClient}
		r.Register(NewSlackSearchTool(readDeps))
		r.Register(NewSlackReadChannelTool(readDeps))
		r.Register(NewSlackReadThreadTool(readDeps))
		writeDeps := SlackToolDeps{Client: deps.SlackClient, Interactor: deny}
		r.Register(NewSlackSendTool(writeDeps))
		r.Register(NewSlackEditTool(writeDeps))
		r.Register(NewSlackReactTool(writeDeps))
	}
	if len(deps.Agents) > 0 {
		r.Register(NewDelegateTaskTool(DelegateTaskConfig{
			Interactor: deny,
			Agents:     deps.Agents,
			ScratchDir: deps.ScratchDir,
		}))
	}

	// NOT registered: subagent (no recursion)
	return r
}
