package provider

import (
	"fmt"
	"strings"

	"github.com/priyanshujain/openbotkit/config"
)

// ProviderEnvVars maps provider names to their conventional env var names.
var ProviderEnvVars = map[string]string{
	"anthropic": "ANTHROPIC_API_KEY",
	"openai":    "OPENAI_API_KEY",
	"gemini":    "GEMINI_API_KEY",
}

// Factory creates a Provider from a model provider config and resolved API key.
type Factory func(cfg config.ModelProviderConfig, apiKey string) Provider

// factories holds registered provider factories.
var factories = map[string]Factory{}

// RegisterFactory registers a provider factory by name.
func RegisterFactory(name string, f Factory) {
	factories[name] = f
}

// Registry manages instantiation of model providers from config.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a registry from the given models config.
func NewRegistry(models *config.ModelsConfig) (*Registry, error) {
	r := &Registry{providers: make(map[string]Provider)}

	if models == nil {
		return r, nil
	}

	// Determine which providers are referenced.
	needed := make(map[string]bool)
	for _, spec := range []string{models.Default, models.Complex, models.Fast} {
		if spec == "" {
			continue
		}
		name, _, err := ParseModelSpec(spec)
		if err != nil {
			return nil, err
		}
		needed[name] = true
	}

	// Instantiate needed providers.
	for name := range needed {
		var providerCfg config.ModelProviderConfig
		if models.Providers != nil {
			providerCfg = models.Providers[name]
		}

		p, err := createProvider(name, providerCfg)
		if err != nil {
			return nil, fmt.Errorf("create provider %q: %w", name, err)
		}
		r.providers[name] = p
	}

	return r, nil
}

// Get returns the provider with the given name.
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

func createProvider(name string, cfg config.ModelProviderConfig) (Provider, error) {
	factory, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q (not registered)", name)
	}

	var apiKey string
	if cfg.AuthMethod != "vertex_ai" {
		envVar := ProviderEnvVars[name]
		var err error
		apiKey, err = ResolveAPIKey(cfg.APIKeyRef, envVar)
		if err != nil {
			return nil, err
		}
	}

	return factory(cfg, apiKey), nil
}

// GetFactory returns the registered factory for the given provider name.
func GetFactory(name string) (Factory, bool) {
	f, ok := factories[name]
	return f, ok
}

// ParseModelSpec splits "provider/model" into provider name and model ID.
// e.g. "anthropic/claude-sonnet-4-6" → ("anthropic", "claude-sonnet-4-6", nil)
func ParseModelSpec(spec string) (providerName, model string, err error) {
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid model spec %q (want provider/model)", spec)
	}
	return parts[0], parts[1], nil
}
