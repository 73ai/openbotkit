package slack

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	slacksrc "github.com/73ai/openbotkit/source/slack"
	"github.com/73ai/openbotkit/source/slack/desktop"
	"github.com/charmbracelet/huh"
)

func SelectDesktopWorkspaces(creds []*desktop.Credentials) ([]*desktop.Credentials, error) {
	if len(creds) <= 1 {
		return creds, nil
	}

	options := make([]huh.Option[int], len(creds))
	for i, c := range creds {
		options[i] = huh.NewOption(fmt.Sprintf("%s (%s)", c.TeamName, c.TeamID), i).Selected(true)
	}

	var picked []int
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Which Slack workspaces should obk connect to?").
				Options(options...).
				Value(&picked),
		),
	).Run()
	if err != nil {
		return nil, err
	}

	if len(picked) == 0 {
		return nil, fmt.Errorf("no Slack workspace selected")
	}
	selected := make([]*desktop.Credentials, 0, len(picked))
	for _, i := range picked {
		selected = append(selected, creds[i])
	}
	return selected, nil
}

// AddDesktopWorkspaces stores credentials and records each workspace in cfg;
// the caller saves cfg.
func AddDesktopWorkspaces(cfg *config.Config, creds []*desktop.Credentials) ([]string, error) {
	var names []string
	for _, c := range creds {
		workspace := slacksrc.SanitizeWorkspaceName(c.TeamName)
		if err := slacksrc.SaveCredentials(workspace, c.Token, c.Cookie); err != nil {
			return nil, fmt.Errorf("save credentials for %s: %w", workspace, err)
		}
		addWorkspace(cfg, workspace, c.TeamID, c.TeamName, "desktop")
		names = append(names, workspace)
	}
	return names, nil
}

func addWorkspace(cfg *config.Config, name, teamID, teamName, authMode string) {
	if cfg.Slack == nil {
		cfg.Slack = &config.SlackConfig{}
	}
	if cfg.Slack.Workspaces == nil {
		cfg.Slack.Workspaces = make(map[string]config.SlackWorkspace)
	}
	cfg.Slack.Workspaces[name] = config.SlackWorkspace{TeamID: teamID, TeamName: teamName, AuthMode: authMode}
	if cfg.Slack.DefaultWorkspace == "" {
		cfg.Slack.DefaultWorkspace = name
	}
}
