package state

import (
	"fmt"
	"strings"
)

type artifactLocation struct {
	sessionID  string
	lineageKey string
	index      int
}

// LoadArtifactByID finds one artifact by globally unique id.
func LoadArtifactByID(artifactID string) (Artifact, error) {
	st, err := Load("")
	if err != nil {
		return Artifact{}, err
	}

	location, err := findUniqueArtifactLocation(st, artifactID)
	if err != nil {
		return Artifact{}, err
	}

	return st.Sessions[location.sessionID].Lineages[location.lineageKey].Artifacts[location.index], nil
}

func findUniqueArtifactLocation(st State, artifactID string) (artifactLocation, error) {
	targetID := strings.TrimSpace(artifactID)
	if targetID == "" {
		return artifactLocation{}, fmt.Errorf("artifact id is required")
	}

	var found *artifactLocation
	for sessionID, session := range st.Sessions {
		for lineageKey, lineage := range session.Lineages {
			for idx, artifact := range lineage.Artifacts {
				if artifact.ID != targetID {
					continue
				}

				current := artifactLocation{sessionID: sessionID, lineageKey: lineageKey, index: idx}
				if found != nil {
					return artifactLocation{}, fmt.Errorf("artifact id %q is not unique", targetID)
				}
				found = &current
			}
		}
	}

	if found == nil {
		return artifactLocation{}, fmt.Errorf("artifact %q not found", targetID)
	}

	return *found, nil
}
