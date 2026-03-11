package tools

import "github.com/priyanshujain/openbotkit/source/slack"

type SlackToolDeps struct {
	Client     slack.API
	Interactor Interactor
	Workspace  string
}
