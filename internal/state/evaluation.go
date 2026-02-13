package state

import (
	"fmt"
	"strings"
	"time"
)

// EvaluateArtifact stores immutable single-score feedback for one artifact.
func EvaluateArtifact(artifactID string, score int, comment string) error {
	if score < 1 || score > 10 {
		return fmt.Errorf("score must be between 1-10")
	}

	st, err := Load("")
	if err != nil {
		return err
	}

	for sessionID, session := range st.Sessions {
		for lineageKey, lineage := range session.Lineages {
			for idx := range lineage.Artifacts {
				if lineage.Artifacts[idx].ID != artifactID {
					continue
				}
				if lineage.Artifacts[idx].Evaluation != nil {
					return fmt.Errorf("artifact already evaluated")
				}

				lineage.Artifacts[idx].Evaluation = &Evaluation{
					Score:       score,
					Comment:     strings.TrimSpace(comment),
					EvaluatedAt: time.Now().UTC().Format(time.RFC3339),
				}

				session.Lineages[lineageKey] = lineage
				st.Sessions[sessionID] = session
				return Save("", st)
			}
		}
	}

	return fmt.Errorf("artifact %q not found", artifactID)
}
