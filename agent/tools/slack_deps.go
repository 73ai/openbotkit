package tools

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/slack"
)

type SlackToolDeps struct {
	Client        slack.API
	Interactor    Interactor
	ApprovalRules *ApprovalRuleSet
	Workspace     string // enables ID -> name resolution
	ScratchDir    string // where downloaded media is written
	// Workspaces follow Client's workspace; when Client is nil the first is the default.
	Workspaces []*SlackWorkspace
}

type SlackWorkspace struct {
	Name      string
	TeamName  string
	Client    slack.API
	Resolver  *slack.Resolver
	UserCache *slack.UserCache
}

func NewSlackWorkspace(name, teamName string, client slack.API) *SlackWorkspace {
	ws := &SlackWorkspace{Name: name, TeamName: teamName, Client: client, Resolver: slack.NewResolver(client)}
	if name != "" {
		ws.UserCache = slack.NewUserCache(name, client)
	}
	return ws
}

type slackRouter []*SlackWorkspace

func (d SlackToolDeps) router() slackRouter {
	if d.Client == nil {
		return d.Workspaces
	}
	return append(slackRouter{NewSlackWorkspace(d.Workspace, "", d.Client)}, d.Workspaces...)
}

func (r slackRouter) pick(name string) (*SlackWorkspace, error) {
	if name == "" {
		return r[0], nil
	}
	name = slack.SanitizeWorkspaceName(strings.TrimSpace(name))
	for _, ws := range r {
		if ws.Name == name {
			return ws, nil
		}
	}
	return nil, fmt.Errorf("unknown Slack workspace %q; connected: %s", name, r.names())
}

func (r slackRouter) forSubdomain(name, subdomain string) (*SlackWorkspace, error) {
	if name != "" {
		return r.pick(name)
	}
	for _, ws := range r {
		if slack.MatchWorkspace(subdomain, ws.Name, ws.TeamName) {
			return ws, nil
		}
	}
	return nil, fmt.Errorf("permalink is from workspace %q, which is not connected; connected: %s", subdomain, r.names())
}

func (r slackRouter) names() string {
	names := make([]string, len(r))
	for i, ws := range r {
		names[i] = ws.Name
	}
	return strings.Join(names, ", ")
}

func (r slackRouter) describe(desc string) string {
	if len(r) < 2 {
		return desc
	}
	return fmt.Sprintf("%s Connected workspaces: %s (default: %s); set workspace to choose another.", desc, r.names(), r[0].Name)
}

func (r slackRouter) channelLabel(ws *SlackWorkspace, channel string) string {
	if len(r) < 2 {
		return channel
	}
	return fmt.Sprintf("%s in %s workspace", channel, ws.Name)
}

const slackWorkspaceProp = `"workspace": {
				"type": "string",
				"description": "Slack workspace name; defaults to the default workspace"
			},`

// LoadSlackToolDeps loads every configured workspace with stored credentials,
// default first. ok is false when the default workspace has none.
func LoadSlackToolDeps(cfg *config.SlackConfig) (deps SlackToolDeps, ok bool) {
	if cfg == nil || cfg.DefaultWorkspace == "" {
		return deps, false
	}
	names := make([]string, 0, len(cfg.Workspaces))
	for name := range cfg.Workspaces {
		if name != cfg.DefaultWorkspace {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	for _, name := range append([]string{cfg.DefaultWorkspace}, names...) {
		creds, err := slack.LoadCredentials(name)
		if err != nil {
			if name == cfg.DefaultWorkspace {
				return deps, false
			}
			slog.Debug("slack workspace skipped: no credentials", "workspace", name)
			continue
		}
		client := slack.NewClient(creds.Token, creds.Cookie)
		deps.Workspaces = append(deps.Workspaces, NewSlackWorkspace(name, cfg.Workspaces[name].TeamName, client))
	}
	return deps, true
}

// truncateUTF8 truncates s to at most maxRunes runes, appending "..." if truncated.
func truncateUTF8(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
