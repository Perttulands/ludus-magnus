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

	location, err := findUniqueArtifactLocation(st, artifactID)
	if err != nil {
		return err
	}

	session := st.Sessions[location.sessionID]
	lineage := session.Lineages[location.lineageKey]
	if lineage.Artifacts[location.index].Evaluation != nil {
		return fmt.Errorf("artifact already evaluated")
	}

	lineage.Artifacts[location.index].Evaluation = &Evaluation{
		Score:       score,
		Comment:     strings.TrimSpace(comment),
		EvaluatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	session.Lineages[location.lineageKey] = lineage
	st.Sessions[location.sessionID] = session
	return Save("", st)
}
