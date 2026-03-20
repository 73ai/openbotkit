package cerebras

import (
	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/provider"
	"github.com/73ai/openbotkit/provider/openai"
)

const defaultBaseURL = "https://api.cerebras.ai"

func init() {
	provider.RegisterFactory("cerebras", func(cfg config.ModelProviderConfig, apiKey string) provider.Provider {
		opts := []openai.Option{openai.WithBaseURL(defaultBaseURL)}
		if cfg.BaseURL != "" {
			opts = []openai.Option{openai.WithBaseURL(cfg.BaseURL)}
		}
		return openai.New(apiKey, opts...)
	})
}
