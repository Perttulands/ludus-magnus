package export

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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
	toolsLiteral := pythonLiteral(def.Tools)
	return fmt.Sprintf(
		"agent_definition = {\n"+
			"    \"system_prompt\": %s,\n"+
			"    \"model\": %s,\n"+
			"    \"temperature\": %g,\n"+
			"    \"max_tokens\": %d,\n"+
			"    \"tools\": %s\n"+
			"}\n",
		pythonString(def.SystemPrompt),
		pythonString(def.Model),
		def.Temperature,
		def.MaxTokens,
		toolsLiteral,
	)
}

func renderTypeScript(def state.AgentDefinition) string {
	toolsLiteral := jsonValue(def.Tools)
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
			"  tools: %s\n"+
			"};\n\n"+
			"export default agentDefinition;\n",
		jsonString(def.SystemPrompt),
		jsonString(def.Model),
		def.Temperature,
		def.MaxTokens,
		toolsLiteral,
	)
}

func jsonString(value string) string {
	payload, err := json.Marshal(value)
	if err != nil {
		// Strings should always marshal, but keep output valid if this ever fails.
		return `""`
	}
	return string(payload)
}

func pythonString(value string) string {
	return jsonString(value)
}

func jsonValue(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(payload)
}

func pythonLiteral(value any) string {
	switch v := value.(type) {
	case nil:
		return "None"
	case bool:
		if v {
			return "True"
		}
		return "False"
	case string:
		return pythonString(v)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, pythonLiteral(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", pythonString(key), pythonLiteral(v[key])))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return jsonValue(v)
	}
}
