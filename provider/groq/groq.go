package groq

import (
	"github.com/priyanshujain/openbotkit/config"
	"github.com/priyanshujain/openbotkit/provider"
	"github.com/priyanshujain/openbotkit/provider/openai"
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
