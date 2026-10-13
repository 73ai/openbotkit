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
	Resolver      *slack.Resolver
	Interactor    Interactor
	ApprovalRules *ApprovalRuleSet
	Workspace     string // enables ID -> name resolution
	ScratchDir    string // where downloaded media is written
	UserCache     *slack.UserCache
	Workspaces    []*SlackWorkspace // other connected workspaces, chosen per call by name
}

// SlackWorkspace is one connected workspace and its lookup caches.
type SlackWorkspace struct {
	Name      string
	TeamName  string
	Client    slack.API
	Resolver  *slack.Resolver
	UserCache *slack.UserCache
}

func NewSlackWorkspace(name, teamName string, client slack.API) *SlackWorkspace {
	return &SlackWorkspace{
		Name:      name,
		TeamName:  teamName,
		Client:    client,
		Resolver:  slack.NewResolver(client),
		UserCache: slack.NewUserCache(name, client),
	}
}

type slackRouter struct {
	def    *SlackWorkspace
	byName map[string]*SlackWorkspace
	names  []string
}

func (d SlackToolDeps) router() *slackRouter {
	def := &SlackWorkspace{
		Name:      d.Workspace,
		Client:    d.Client,
		Resolver:  d.SlackResolver(),
		UserCache: d.SlackUserCache(),
	}
	r := &slackRouter{def: def, byName: map[string]*SlackWorkspace{}}
	if def.Name != "" {
		r.byName[def.Name] = def
		r.names = append(r.names, def.Name)
	}
	for _, ws := range d.Workspaces {
		if _, dup := r.byName[ws.Name]; dup || ws.Name == "" {
			continue
		}
		r.byName[ws.Name] = ws
		r.names = append(r.names, ws.Name)
	}
	return r
}

// pick returns the named workspace, or the default when name is empty.
func (r *slackRouter) pick(name string) (*SlackWorkspace, error) {
	if name == "" {
		return r.def, nil
	}
	if ws, ok := r.byName[slack.SanitizeWorkspaceName(strings.TrimSpace(name))]; ok {
		return ws, nil
	}
	return nil, fmt.Errorf("unknown Slack workspace %q; connected: %s", name, strings.Join(r.names, ", "))
}

// forSubdomain picks the named workspace, else the one matching a permalink
// subdomain, else the default.
func (r *slackRouter) forSubdomain(name, subdomain string) (*SlackWorkspace, error) {
	if name != "" || subdomain == "" || len(r.names) < 2 {
		return r.pick(name)
	}
	for _, n := range r.names {
		ws := r.byName[n]
		if slack.MatchesWorkspace(subdomain, ws.Name) || (ws.TeamName != "" && slack.MatchesWorkspace(subdomain, ws.TeamName)) {
			return ws, nil
		}
	}
	return r.def, nil
}

// describe appends the connected workspaces to a tool description when there
// is more than one to choose from.
func (r *slackRouter) describe(desc string) string {
	if len(r.names) < 2 {
		return desc
	}
	return fmt.Sprintf("%s Connected workspaces: %s (default: %s); set workspace to choose another.", desc, strings.Join(r.names, ", "), r.def.Name)
}

const slackWorkspaceProp = `"workspace": {
				"type": "string",
				"description": "Slack workspace name; defaults to the default workspace"
			},`

// SlackResolver returns the shared Resolver, creating one if needed.
func (d SlackToolDeps) SlackResolver() *slack.Resolver {
	if d.Resolver != nil {
		return d.Resolver
	}
	return slack.NewResolver(d.Client)
}

// SlackUserCache returns the name cache, or nil when no workspace is
// configured — callers treat nil as "skip name resolution".
func (d SlackToolDeps) SlackUserCache() *slack.UserCache {
	if d.UserCache != nil {
		return d.UserCache
	}
	if d.Workspace == "" {
		return nil
	}
	return slack.NewUserCache(d.Workspace, d.Client)
}

// truncateUTF8 truncates s to at most maxRunes runes, appending "..." if truncated.
func truncateUTF8(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

// channelLabel names a channel for approval prompts, qualified by workspace
// when several are connected.
func (r *slackRouter) channelLabel(ws *SlackWorkspace, channel string) string {
	if len(r.names) < 2 {
		return channel
	}
	return fmt.Sprintf("%s in %s workspace", channel, ws.Name)
}

// LoadSlackToolDeps builds deps for the default workspace plus every other
// configured workspace with stored credentials. ok is false when the default
// workspace has no credentials.
func LoadSlackToolDeps(cfg *config.SlackConfig) (deps SlackToolDeps, ok bool) {
	return loadSlackToolDeps(cfg, slack.LoadCredentials)
}

func loadSlackToolDeps(cfg *config.SlackConfig, load func(string) (*slack.Credentials, error)) (SlackToolDeps, bool) {
	if cfg == nil || cfg.DefaultWorkspace == "" {
		return SlackToolDeps{}, false
	}
	creds, err := load(cfg.DefaultWorkspace)
	if err != nil {
		return SlackToolDeps{}, false
	}
	deps := SlackToolDeps{
		Client:    slack.NewClient(creds.Token, creds.Cookie),
		Workspace: cfg.DefaultWorkspace,
	}

	names := make([]string, 0, len(cfg.Workspaces))
	for name := range cfg.Workspaces {
		if name != cfg.DefaultWorkspace {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		creds, err := load(name)
		if err != nil {
			slog.Debug("slack workspace skipped: no credentials", "workspace", name)
			continue
		}
		client := slack.NewClient(creds.Token, creds.Cookie)
		deps.Workspaces = append(deps.Workspaces, NewSlackWorkspace(name, cfg.Workspaces[name].TeamName, client))
	}
	return deps, true
}
