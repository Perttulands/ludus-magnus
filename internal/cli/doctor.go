package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type doctorCheck struct {
	Name   string
	Status string
	Detail string
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run environment diagnostics",
		RunE: func(_ *cobra.Command, _ []string) error {
			checks := []doctorCheck{
				checkGoVersion(),
				checkLLMConfig(),
				checkBeads(),
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Check", "Status", "Details"})

			for _, check := range checks {
				table.Append([]string{check.Name, stylizeStatus(check.Status), check.Detail})
			}

			table.Render()
			return nil
		},
	}
}

func checkGoVersion() doctorCheck {
	goBinary := resolveGoBinary()
	if goBinary == "" {
		return doctorCheck{Name: "Go", Status: "FAIL", Detail: "go binary not found in PATH or /usr/local/go/bin/go"}
	}

	out, err := exec.Command(goBinary, "version").CombinedOutput()
	if err != nil {
		return doctorCheck{Name: "Go", Status: "FAIL", Detail: fmt.Sprintf("failed to run '%s version': %v", goBinary, err)}
	}

	return doctorCheck{Name: "Go", Status: "PASS", Detail: strings.TrimSpace(string(out))}
}

func checkLLMConfig() doctorCheck {
	keys := []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GOOGLE_API_KEY", "MISTRAL_API_KEY", "OLLAMA_HOST"}
	configured := make([]string, 0, len(keys))

	for _, key := range keys {
		if value := strings.TrimSpace(viper.GetString(key)); value != "" {
			configured = append(configured, key)
		}
	}

	if len(configured) == 0 {
		return doctorCheck{Name: "LLM Config", Status: "WARN", Detail: "no known LLM environment variables are set"}
	}

	return doctorCheck{Name: "LLM Config", Status: "PASS", Detail: fmt.Sprintf("configured keys: %s", strings.Join(configured, ", "))}
}

func checkBeads() doctorCheck {
	if path, err := exec.LookPath("br"); err == nil {
		return doctorCheck{Name: "Beads", Status: "PASS", Detail: fmt.Sprintf("br found at %s", path)}
	}
	if path, err := exec.LookPath("bd"); err == nil {
		return doctorCheck{Name: "Beads", Status: "WARN", Detail: fmt.Sprintf("legacy bd found at %s; migrate to br", path)}
	}

	return doctorCheck{Name: "Beads", Status: "WARN", Detail: "neither br nor bd command found"}
}

func resolveGoBinary() string {
	if path, err := exec.LookPath("go"); err == nil {
		return path
	}
	if _, err := os.Stat("/usr/local/go/bin/go"); err == nil {
		return "/usr/local/go/bin/go"
	}

	return ""
}

func stylizeStatus(status string) string {
	switch status {
	case "PASS":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(status)
	case "WARN":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(status)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(status)
	}
}
