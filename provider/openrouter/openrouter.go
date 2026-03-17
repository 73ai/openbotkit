package openrouter

import (
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/provider/openai"
)

const defaultBaseURL = "https://openrouter.ai/api"

func init() {
	provider.RegisterFactory("openrouter", func(cfg config.ModelProviderConfig, apiKey string) provider.Provider {
		opts := []openai.Option{openai.WithBaseURL(defaultBaseURL)}
		if cfg.BaseURL != "" {
			opts = []openai.Option{openai.WithBaseURL(cfg.BaseURL)}
		}
		return openai.New(apiKey, opts...)
	})
}
