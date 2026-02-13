package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

const (
	FormatJSON       = "json"
	FormatPython     = "python"
	FormatTypeScript = "typescript"
)

// AgentDefinition renders one stored agent definition in the requested format.
func AgentDefinition(st state.State, agentID string, format string) (string, error) {
	agent, err := findUniqueAgentByID(st, agentID)
	if err != nil {
		return "", err
	}

	switch normalizeFormat(format) {
	case FormatJSON:
		return renderJSON(agent.Definition)
	case FormatPython:
		return renderPython(agent.Definition), nil
	case FormatTypeScript:
		return renderTypeScript(agent.Definition), nil
	default:
		return "", fmt.Errorf("unsupported export format %q", format)
	}
}

func findUniqueAgentByID(st state.State, agentID string) (state.Agent, error) {
	targetID := strings.TrimSpace(agentID)
	if targetID == "" {
		return state.Agent{}, fmt.Errorf("agent id is required")
	}

	var found *state.Agent
	for _, session := range st.Sessions {
		for _, lineage := range session.Lineages {
			for _, agent := range lineage.Agents {
				if agent.ID != targetID {
					continue
				}
				if found != nil {
					return state.Agent{}, fmt.Errorf("agent id %q is not unique", targetID)
				}
				candidate := agent
				found = &candidate
			}
		}
	}

	if found == nil {
		return state.Agent{}, fmt.Errorf("agent %q not found", targetID)
	}

	return *found, nil
}

func normalizeFormat(format string) string {
	trimmed := strings.TrimSpace(format)
	if trimmed == "" {
		return FormatJSON
	}
	return strings.ToLower(trimmed)
}

func renderJSON(def state.AgentDefinition) (string, error) {
	payload, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal agent definition: %w", err)
	}
	return string(payload) + "\n", nil
}

func renderPython(def state.AgentDefinition) string {
	return fmt.Sprintf(
		"agent_definition = {\n"+
			"    \"system_prompt\": %s,\n"+
			"    \"model\": %s,\n"+
			"    \"temperature\": %g,\n"+
			"    \"max_tokens\": %d,\n"+
			"    \"tools\": []\n"+
			"}\n",
		pythonString(def.SystemPrompt),
		pythonString(def.Model),
		def.Temperature,
		def.MaxTokens,
	)
}

func renderTypeScript(def state.AgentDefinition) string {
	return fmt.Sprintf(
		"type AgentDefinition = {\n"+
			"  systemPrompt: string;\n"+
			"  model: string;\n"+
			"  temperature: number;\n"+
			"  maxTokens: number;\n"+
			"  tools: unknown[];\n"+
			"};\n\n"+
			"const agentDefinition: AgentDefinition = {\n"+
			"  systemPrompt: %s,\n"+
			"  model: %s,\n"+
			"  temperature: %g,\n"+
			"  maxTokens: %d,\n"+
			"  tools: []\n"+
			"};\n\n"+
			"export default agentDefinition;\n",
		jsonString(def.SystemPrompt),
		jsonString(def.Model),
		def.Temperature,
		def.MaxTokens,
	)
}

func jsonString(value string) string {
	payload, _ := json.Marshal(value)
	return string(payload)
}

func pythonString(value string) string {
	return jsonString(value)
}
