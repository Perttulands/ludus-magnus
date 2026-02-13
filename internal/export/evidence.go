package export

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

// EvidencePack renders one session's evidence bundle as JSON.
func EvidencePack(st state.State, sessionID string, format string) (string, error) {
	targetID := strings.TrimSpace(sessionID)
	if targetID == "" {
		return "", fmt.Errorf("session id is required")
	}

	session, ok := st.Sessions[targetID]
	if !ok {
		return "", fmt.Errorf("session %q not found", targetID)
	}

	switch normalizeFormat(format) {
	case FormatJSON:
		pack := buildEvidencePack(session)
		payload, err := json.MarshalIndent(pack, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal evidence pack: %w", err)
		}
		return string(payload) + "\n", nil
	default:
		return "", fmt.Errorf("unsupported export format %q", format)
	}
}

type evidencePack struct {
	SessionID string            `json:"session_id"`
	Mode      string            `json:"mode"`
	Need      string            `json:"need"`
	CreatedAt string            `json:"created_at"`
	Lineages  []evidenceLineage `json:"lineages"`
}

type evidenceLineage struct {
	Name          string           `json:"name"`
	Locked        bool             `json:"locked"`
	AgentVersions []evidenceAgent  `json:"agent_versions"`
	Artifacts     []state.Artifact `json:"artifacts"`
	Directives    state.Directives `json:"directives"`
}

type evidenceAgent struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	SystemPrompt string `json:"system_prompt"`
	CreatedAt    string `json:"created_at"`
}

func buildEvidencePack(session state.Session) evidencePack {
	lineageIDs := make([]string, 0, len(session.Lineages))
	for lineageID := range session.Lineages {
		lineageIDs = append(lineageIDs, lineageID)
	}
	sort.Strings(lineageIDs)

	lineages := make([]evidenceLineage, 0, len(lineageIDs))
	for _, lineageID := range lineageIDs {
		lineage := session.Lineages[lineageID]
		agentVersions := make([]evidenceAgent, 0, len(lineage.Agents))
		for _, agent := range lineage.Agents {
			agentVersions = append(agentVersions, evidenceAgent{
				ID:           agent.ID,
				Version:      agent.Version,
				SystemPrompt: agent.Definition.SystemPrompt,
				CreatedAt:    agent.CreatedAt,
			})
		}

		lineages = append(lineages, evidenceLineage{
			Name:          lineage.Name,
			Locked:        lineage.Locked,
			AgentVersions: agentVersions,
			Artifacts:     lineage.Artifacts,
			Directives:    lineage.Directives,
		})
	}

	return evidencePack{
		SessionID: session.ID,
		Mode:      session.Mode,
		Need:      session.Need,
		CreatedAt: session.CreatedAt,
		Lineages:  lineages,
	}
}
