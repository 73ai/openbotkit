package slack

import (
	"fmt"

	"github.com/73ai/openbotkit/config"
	slacksrc "github.com/73ai/openbotkit/source/slack"
	"github.com/73ai/openbotkit/source/slack/desktop"
	"github.com/charmbracelet/huh"
)

// SelectDesktopWorkspaces asks which of the detected workspaces to connect.
// All are selected by default.
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

	selected := make([]*desktop.Credentials, 0, len(picked))
	for _, i := range picked {
		selected = append(selected, creds[i])
	}
	return selected, nil
}

// AddDesktopWorkspaces stores credentials for each workspace and records it in
// cfg. The caller saves cfg. Returns the workspace names added.
func AddDesktopWorkspaces(cfg *config.Config, creds []*desktop.Credentials) ([]string, error) {
	if cfg.Slack == nil {
		cfg.Slack = &config.SlackConfig{}
	}
	if cfg.Slack.Workspaces == nil {
		cfg.Slack.Workspaces = make(map[string]config.SlackWorkspace)
	}

	var names []string
	for _, c := range creds {
		workspace := slacksrc.SanitizeWorkspaceName(c.TeamName)
		if err := slacksrc.SaveCredentials(workspace, c.Token, c.Cookie); err != nil {
			return names, fmt.Errorf("save credentials for %s: %w", workspace, err)
		}
		cfg.Slack.Workspaces[workspace] = config.SlackWorkspace{
			TeamID:   c.TeamID,
			TeamName: c.TeamName,
			AuthMode: "desktop",
		}
		if cfg.Slack.DefaultWorkspace == "" {
			cfg.Slack.DefaultWorkspace = workspace
		}
		names = append(names, workspace)
	}
	return names, nil
}
