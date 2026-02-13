package provider

import (
	"fmt"
	"os"
	"strings"
)

// Config configures provider selection using flags/env values.
type Config struct {
	Provider string
	Model    string
	BaseURL  string
	APIKey   string
}

// NewFactory builds a provider adapter from config and environment.
func NewFactory(cfg Config) (Provider, error) {
	providerName := normalizeProviderName(cfg.Provider)

	switch providerName {
	case "anthropic":
		key := firstNonEmpty(strings.TrimSpace(cfg.APIKey), os.Getenv("ANTHROPIC_API_KEY"))
		if key == "" {
			return nil, fmt.Errorf("missing anthropic credentials: set ANTHROPIC_API_KEY")
		}
		return NewAnthropicProvider(key, cfg.Model, cfg.BaseURL), nil
	case "openai-compatible":
		key := firstNonEmpty(
			strings.TrimSpace(cfg.APIKey),
			os.Getenv("OPENAI_API_KEY"),
			os.Getenv("OPENAI_COMPATIBLE_API_KEY"),
			os.Getenv("API_KEY"),
		)
		if key == "" {
			return nil, fmt.Errorf("missing openai-compatible credentials: set OPENAI_API_KEY or equivalent")
		}
		return NewOpenAICompatibleProvider(key, cfg.Model, cfg.BaseURL), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}

func normalizeProviderName(raw string) string {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return "anthropic"
	}
	switch name {
	case "openai", "openai_compatible", "openrouter", "litellm":
		return "openai-compatible"
	default:
		return name
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
