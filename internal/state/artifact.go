package state

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AddArtifact appends one artifact to a lineage in the default state file.
func AddArtifact(sessionID, lineageID string, artifact Artifact) (string, error) {
	st, err := Load("")
	if err != nil {
		return "", err
	}

	session, ok := st.Sessions[sessionID]
	if !ok {
		return "", fmt.Errorf("session %q not found", sessionID)
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
		return "", fmt.Errorf("lineage %q not found in session %q", lineageID, sessionID)
	}

	if strings.TrimSpace(artifact.ID) == "" {
		artifact.ID, err = newUniqueArtifactID(st)
		if err != nil {
			return "", err
		}
	} else if artifactIDExists(st, artifact.ID) {
		return "", fmt.Errorf("artifact id %q already exists", artifact.ID)
	}
	if strings.TrimSpace(artifact.CreatedAt) == "" {
		artifact.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	lineage.Artifacts = append(lineage.Artifacts, artifact)
	session.Lineages[lineageKey] = lineage
	st.Sessions[sessionID] = session

	if err := Save("", st); err != nil {
		return "", err
	}
	return artifact.ID, nil
}

func newArtifactID() string {
	return fmt.Sprintf("art_%s", strings.ReplaceAll(uuid.NewString(), "-", "")[:8])
}

func newUniqueArtifactID(st State) (string, error) {
	const maxAttempts = 256
	for i := 0; i < maxAttempts; i++ {
		candidate := newArtifactID()
		if !artifactIDExists(st, candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("failed to generate globally unique artifact id after %d attempts", maxAttempts)
}

func artifactIDExists(st State, artifactID string) bool {
	for _, session := range st.Sessions {
		for _, lineage := range session.Lineages {
			for _, artifact := range lineage.Artifacts {
				if artifact.ID == artifactID {
					return true
				}
			}
		}
	}
	return false
}
