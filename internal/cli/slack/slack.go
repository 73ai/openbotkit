package slack

import (
	"fmt"
	"sort"
	"strings"

	"github.com/73ai/openbotkit/config"
	slacksrc "github.com/73ai/openbotkit/source/slack"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "slack",
	Short: "Manage Slack data source",
}

var workspaceFlag string

func loadClient() (*slacksrc.Client, error) {
	client, _, _, err := loadClientAndCache("")
	return client, err
}

// loadClientAndCache also returns the user name cache and the workspace it is
// keyed on, for commands that resolve IDs to names. subdomain, taken from a
// permalink, picks the matching workspace when --workspace is not set.
func loadClientAndCache(subdomain string) (*slacksrc.Client, *slacksrc.UserCache, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, "", fmt.Errorf("load config: %w", err)
	}
	workspace, err := resolveWorkspace(cfg.Slack, workspaceFlag, subdomain)
	if err != nil {
		return nil, nil, "", err
	}
	creds, err := slacksrc.LoadCredentials(workspace)
	if err != nil {
		return nil, nil, "", fmt.Errorf("load credentials: %w", err)
	}
	client := slacksrc.NewClient(creds.Token, creds.Cookie)
	return client, slacksrc.NewUserCache(workspace, client), workspace, nil
}

func resolveWorkspace(cfg *config.SlackConfig, explicit, subdomain string) (string, error) {
	if cfg == nil || len(cfg.Workspaces) == 0 {
		return "", fmt.Errorf("no Slack workspace configured; run: obk slack auth login")
	}
	if explicit != "" {
		name := slacksrc.SanitizeWorkspaceName(explicit)
		if _, ok := cfg.Workspaces[name]; !ok {
			return "", fmt.Errorf("unknown Slack workspace %q; configured: %s", explicit, strings.Join(workspaceNames(cfg), ", "))
		}
		return name, nil
	}
	if subdomain != "" {
		for _, name := range workspaceNames(cfg) {
			ws := cfg.Workspaces[name]
			if slacksrc.MatchesWorkspace(subdomain, name) || (ws.TeamName != "" && slacksrc.MatchesWorkspace(subdomain, ws.TeamName)) {
				return name, nil
			}
		}
	}
	if cfg.DefaultWorkspace == "" {
		return "", fmt.Errorf("no default Slack workspace; pass --workspace (configured: %s)", strings.Join(workspaceNames(cfg), ", "))
	}
	return cfg.DefaultWorkspace, nil
}

func workspaceNames(cfg *config.SlackConfig) []string {
	names := make([]string, 0, len(cfg.Workspaces))
	for name := range cfg.Workspaces {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func init() {
	Cmd.PersistentFlags().StringVarP(&workspaceFlag, "workspace", "w", "", "Slack workspace to use (default: the configured default)")
	Cmd.AddCommand(authCmd)
	Cmd.AddCommand(searchCmd)
	Cmd.AddCommand(channelsCmd)
	Cmd.AddCommand(readCmd)
	Cmd.AddCommand(threadCmd)
	Cmd.AddCommand(mediaCmd)
	Cmd.AddCommand(replyCmd)
}
