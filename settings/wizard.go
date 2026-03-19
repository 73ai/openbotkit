package settings

import (
	"github.com/charmbracelet/huh"
)

func runProfileSelect(selected *string, options []Option) error {
	var huhOpts []huh.Option[string]
	for _, o := range options {
		huhOpts = append(huhOpts, huh.NewOption(o.Label, o.Value))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How would you like to configure models?").
				DescriptionFunc(func() string {
					return ProfilePreview(*selected)
				}, selected).
				Options(huhOpts...).
				Value(selected),
		),
	).Run()
}

func promptAPIKey(providerName string, hasExisting bool) (string, error) {
	var apiKey string
	placeholder := ""
	if hasExisting {
		placeholder = "(leave blank to keep existing)"
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter your " + providerName + " API key").
				Placeholder(placeholder).
				EchoMode(huh.EchoModePassword).
				Value(&apiKey),
		),
	).Run()
	return apiKey, err
}

func selectProvider() (string, error) {
	providers := []Option{
		{"Anthropic", "anthropic"},
		{"OpenAI", "openai"},
		{"Gemini", "gemini"},
		{"Groq", "groq"},
		{"OpenRouter", "openrouter"},
	}

	var selected string
	var opts []huh.Option[string]
	for _, p := range providers {
		opts = append(opts, huh.NewOption(p.Label, p.Value))
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a provider to configure").
				Options(opts...).
				Value(&selected),
		),
	).Run()
	return selected, err
}

func selectModel(title string, options []Option, current string) (string, error) {
	var selected string
	if current != "" {
		selected = current
	}

	var opts []huh.Option[string]
	for _, o := range options {
		opts = append(opts, huh.NewOption(o.Label, o.Value))
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(title).
				Options(opts...).
				Value(&selected),
		),
	).Run()
	return selected, err
}
