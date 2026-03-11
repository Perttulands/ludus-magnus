package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Perttulands/chiron/internal/provider"
	"github.com/Perttulands/chiron/internal/state"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Required bool   `json:"required"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
}

func newDoctorCmd() *cobra.Command {
	var providerName string
	var model string
	var baseURL string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run environment diagnostics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks := []doctorCheck{}
			checks = append(checks, checkProviderCredentials(providerName, apiKey))
			checks = append(checks, checkProviderInitialization(providerName, model, baseURL, apiKey))
			checks = append(checks, checkStateFileReadable())
			checks = append(checks, checkOptionalExecutor("claude"))
			checks = append(checks, checkOptionalExecutor("codex"))

			hasRequiredFailures := false
			if isJSONOutput(cmd) {
				for _, check := range checks {
					if check.Required && !check.Passed {
						hasRequiredFailures = true
					}
				}
				if err := writeJSON(cmd, map[string]any{"checks": checks}); err != nil {
					return fmt.Errorf("marshal results: %w", err)
				}
				if hasRequiredFailures {
					return fmt.Errorf("doctor found failed required checks")
				}
				return nil
			}

			for _, check := range checks {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), check.Message); err != nil {
					return fmt.Errorf("write check result: %w", err)
				}
				if check.Required && !check.Passed {
					hasRequiredFailures = true
				}
			}

			if hasRequiredFailures {
				return fmt.Errorf("doctor found failed required checks")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&providerName, "provider", "anthropic", "Provider name (anthropic, openai-compatible, claude-cli, ollama-native, or pi-cli)")
	cmd.Flags().StringVar(&model, "model", "", "Provider model override")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Provider base URL override")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Provider API key override")

	return cmd
}

func checkProviderCredentials(providerName string, apiKey string) doctorCheck {
	normalized := normalizeDoctorProvider(providerName)
	suppliedAPIKey := strings.TrimSpace(apiKey)

	switch normalized {
	case "anthropic":
		envKey := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
		if suppliedAPIKey != "" || envKey != "" {
			return doctorCheck{Required: true, Passed: true, Message: "✓ ANTHROPIC_API_KEY set"}
		}
		return doctorCheck{Required: true, Passed: false, Message: "✗ missing ANTHROPIC_API_KEY for provider anthropic"}
	case "openai-compatible":
		openaiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		compatKey := strings.TrimSpace(os.Getenv("OPENAI_COMPATIBLE_API_KEY"))
		genericKey := strings.TrimSpace(os.Getenv("API_KEY"))
		if suppliedAPIKey != "" || openaiKey != "" || compatKey != "" || genericKey != "" {
			return doctorCheck{Required: true, Passed: true, Message: "✓ OPENAI_API_KEY (or equivalent) set"}
		}
		return doctorCheck{Required: true, Passed: false, Message: "✗ missing OPENAI_API_KEY (or equivalent) for provider openai-compatible"}
	case "claude-cli":
		_, err := exec.LookPath("claude")
		if err != nil {
			return doctorCheck{Required: true, Passed: false, Message: "✗ claude binary not found (required for claude-cli provider)"}
		}
		return doctorCheck{Required: true, Passed: true, Message: "✓ claude binary found (claude-cli provider uses Claude Code auth)"}
	case "ollama-native":
		return doctorCheck{Required: false, Passed: true, Message: "✓ ollama-native uses HTTP endpoint, no API key required"}
	case "pi-cli":
		_, err := exec.LookPath("pi")
		if err != nil {
			return doctorCheck{Required: true, Passed: false, Message: "✗ pi binary not found (required for pi-cli provider)"}
		}
		return doctorCheck{Required: true, Passed: true, Message: "✓ pi binary found (pi-cli provider uses local Ollama)"}
	default:
		return doctorCheck{Required: true, Passed: false, Message: fmt.Sprintf("✗ unsupported provider: %s", strings.TrimSpace(providerName))}
	}
}

func checkProviderInitialization(providerName string, model string, baseURL string, apiKey string) doctorCheck {
	_, err := provider.NewFactory(provider.Config{
		Provider: providerName,
		Model:    model,
		BaseURL:  baseURL,
		APIKey:   apiKey,
	})
	if err != nil {
		return doctorCheck{Required: true, Passed: false, Message: fmt.Sprintf("✗ Provider initialization failed: %v", err)}
	}
	return doctorCheck{Required: true, Passed: true, Message: fmt.Sprintf("✓ Provider initialization ok: %s", normalizeDoctorProvider(providerName))}
}

func checkStateFileReadable() doctorCheck {
	path := state.DefaultStatePath()
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return doctorCheck{Required: false, Passed: true, Message: fmt.Sprintf("✓ State file not found (optional): %s", path)}
		}
		fmt.Fprintf(os.Stderr, "doctor: state file stat error: %s: %v\n", path, err) //nolint:errcheck // best-effort stderr log
		return doctorCheck{Required: true, Passed: false, Message: fmt.Sprintf("✗ State file stat failed: %s (%v)", path, err)}
	}

	if _, err := os.ReadFile(path); err != nil {
		return doctorCheck{Required: true, Passed: false, Message: fmt.Sprintf("✗ State file not readable: %s (%v)", path, err)}
	}

	return doctorCheck{Required: true, Passed: true, Message: fmt.Sprintf("✓ State file readable: %s", path)}
}

func checkOptionalExecutor(name string) doctorCheck {
	path, err := exec.LookPath(name)
	if err != nil {
		return doctorCheck{Required: false, Passed: true, Message: fmt.Sprintf("✓ %s binary not found (optional)", name)}
	}
	return doctorCheck{Required: false, Passed: true, Message: fmt.Sprintf("✓ %s binary found (optional): %s", name, path)}
}

func normalizeDoctorProvider(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "anthropic"
	}
	if normalized == "openai" || normalized == "openai_compatible" || normalized == "openrouter" || normalized == "litellm" {
		return "openai-compatible"
	}
	if normalized == "claude" || normalized == "claude_cli" || normalized == "claude-code" {
		return "claude-cli"
	}
	if normalized == "pi" || normalized == "pi_cli" || normalized == "pi-cli" || normalized == "pi-ollama" {
		return "pi-cli"
	}
	if normalized == "ollama" || normalized == "ollama-native" || normalized == "ollama_native" {
		return "ollama-native"
	}
	return normalized
}
