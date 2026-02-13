package state

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AddArtifact appends one artifact to a lineage in the default state file.
func AddArtifact(sessionID, lineageID string, artifact Artifact) error {
	st, err := Load("")
	if err != nil {
		return err
	}

	session, ok := st.Sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	lineageKey := ""
	lineage := Lineage{}
	for key, candidate := range session.Lineages {
		if candidate.ID == lineageID {
			lineageKey = key
			lineage = candidate
			break
		}
	}
	if lineageKey == "" {
		return fmt.Errorf("lineage %q not found in session %q", lineageID, sessionID)
	}

	if strings.TrimSpace(artifact.ID) == "" {
		artifact.ID = newArtifactID()
	}
	if strings.TrimSpace(artifact.CreatedAt) == "" {
		artifact.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	lineage.Artifacts = append(lineage.Artifacts, artifact)
	session.Lineages[lineageKey] = lineage
	st.Sessions[sessionID] = session

	return Save("", st)
}

func newArtifactID() string {
	return fmt.Sprintf("art_%s", strings.ReplaceAll(uuid.NewString(), "-", "")[:8])
}
