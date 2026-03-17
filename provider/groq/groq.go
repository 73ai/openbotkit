package groq

import (
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/provider/openai"
)

const defaultBaseURL = "https://api.groq.com/openai"

func init() {
	provider.RegisterFactory("groq", func(cfg config.ModelProviderConfig, apiKey string) provider.Provider {
		opts := []openai.Option{openai.WithBaseURL(defaultBaseURL)}
		if cfg.BaseURL != "" {
			opts = []openai.Option{openai.WithBaseURL(cfg.BaseURL)}
		}
		return openai.New(apiKey, opts...)
	})
}
