package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/73ai/openbotkit/config"
)

// AvailableModel describes a model returned by a provider's list-models API.
type AvailableModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Provider    string `json:"provider"`
}

// ListModels calls the provider's list-models API to enumerate available models.
// This is a free API call that validates the API key and returns the model catalog.
func ListModels(ctx context.Context, providerName, apiKey string, cfg config.ModelProviderConfig) ([]AvailableModel, error) {
	switch providerName {
	case "openai":
		return listModelsOpenAI(ctx, apiKey, cfg)
	case "anthropic":
		return listModelsAnthropic(ctx, apiKey, cfg)
	case "gemini":
		return listModelsGemini(ctx, apiKey, cfg)
	case "groq":
		return listModelsGroq(ctx, apiKey, cfg)
	case "openrouter":
		return listModelsOpenRouter(ctx, apiKey, cfg)
	default:
		return nil, fmt.Errorf("unknown provider %q", providerName)
	}
}

// VerifyModelAccess sends a minimal chat to verify the API key has access to a specific model.
func VerifyModelAccess(ctx context.Context, providerName, modelID, apiKey string, cfg config.ModelProviderConfig) error {
	factory, ok := GetFactory(providerName)
	if !ok {
		return fmt.Errorf("unknown provider %q", providerName)
	}
	p := factory(cfg, apiKey)
	_, err := p.Chat(ctx, ChatRequest{
		Model:     modelID,
		System:    "Reply OK",
		Messages:  []Message{NewTextMessage(RoleUser, "hi")},
		MaxTokens: 5,
	})
	return err
}

func baseURL(cfg config.ModelProviderConfig, defaultURL string) string {
	if cfg.BaseURL != "" {
		return cfg.BaseURL
	}
	return defaultURL
}

// --- OpenAI ---

func listModelsOpenAI(ctx context.Context, apiKey string, cfg config.ModelProviderConfig) ([]AvailableModel, error) {
	url := baseURL(cfg, "https://api.openai.com") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai list models: HTTP %d", resp.StatusCode)
	}

	var body struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	var models []AvailableModel
	for _, m := range body.Data {
		if strings.Contains(m.ID, "gpt") || strings.Contains(m.ID, "o1") || strings.Contains(m.ID, "o3") || strings.Contains(m.ID, "o4") {
			models = append(models, AvailableModel{ID: m.ID, DisplayName: m.ID, Provider: "openai"})
		}
	}
	return models, nil
}

// --- Anthropic ---

func listModelsAnthropic(ctx context.Context, apiKey string, cfg config.ModelProviderConfig) ([]AvailableModel, error) {
	var all []AvailableModel
	afterID := ""
	for {
		url := baseURL(cfg, "https://api.anthropic.com") + "/v1/models?limit=100"
		if afterID != "" {
			url += "&after_id=" + afterID
		}
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("anthropic list models: HTTP %d", resp.StatusCode)
		}

		var body struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
			} `json:"data"`
			HasMore bool `json:"has_more"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, err
		}

		for _, m := range body.Data {
			name := m.DisplayName
			if name == "" {
				name = m.ID
			}
			all = append(all, AvailableModel{ID: m.ID, DisplayName: name, Provider: "anthropic"})
		}

		if !body.HasMore || len(body.Data) == 0 {
			break
		}
		afterID = body.Data[len(body.Data)-1].ID
	}
	return all, nil
}

// --- Gemini ---

func listModelsGemini(ctx context.Context, apiKey string, cfg config.ModelProviderConfig) ([]AvailableModel, error) {
	url := baseURL(cfg, "https://generativelanguage.googleapis.com") + "/v1beta/models?key=" + apiKey
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini list models: HTTP %d", resp.StatusCode)
	}

	var body struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	var models []AvailableModel
	for _, m := range body.Models {
		supportsChat := false
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				supportsChat = true
				break
			}
		}
		if !supportsChat {
			continue
		}
		id := strings.TrimPrefix(m.Name, "models/")
		models = append(models, AvailableModel{ID: id, DisplayName: m.DisplayName, Provider: "gemini"})
	}
	return models, nil
}

// --- Groq (OpenAI-compatible) ---

func listModelsGroq(ctx context.Context, apiKey string, cfg config.ModelProviderConfig) ([]AvailableModel, error) {
	return listModelsOpenAICompat(ctx, apiKey, cfg, "https://api.groq.com/openai", "groq")
}

// --- OpenRouter (OpenAI-compatible) ---

func listModelsOpenRouter(ctx context.Context, apiKey string, cfg config.ModelProviderConfig) ([]AvailableModel, error) {
	return listModelsOpenAICompat(ctx, apiKey, cfg, "https://openrouter.ai/api", "openrouter")
}

func listModelsOpenAICompat(ctx context.Context, apiKey string, cfg config.ModelProviderConfig, defaultBase, providerName string) ([]AvailableModel, error) {
	url := baseURL(cfg, defaultBase) + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s list models: HTTP %d", providerName, resp.StatusCode)
	}

	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	var models []AvailableModel
	for _, m := range body.Data {
		models = append(models, AvailableModel{ID: m.ID, DisplayName: m.ID, Provider: providerName})
	}
	return models, nil
}
