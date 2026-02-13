package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

// GenerateEvolutionPrompt synthesizes artifact evaluations and directives into
// a structured prompt used to produce the next agent version.
func GenerateEvolutionPrompt(agents []state.Agent, artifacts []state.Artifact, directives []state.Directive) string {
	currentVersion, currentSystemPrompt := latestAgentPrompt(agents)
	evaluated := evaluatedArtifacts(artifacts)
	totalArtifacts := len(artifacts)

	avgScore := "N/A"
	scoreHistogram := "No evaluation yet"
	feedbackList := "- No evaluation yet. Use current prompt and directives as baseline improvements."
	lowPatterns := "- None yet"
	highPatterns := "- None yet"

	if len(evaluated) > 0 {
		total := 0
		histogram := map[int]int{}
		feedbackLines := make([]string, 0, len(evaluated))
		lowLines := []string{}
		highLines := []string{}

		for _, artifact := range evaluated {
			score := artifact.Evaluation.Score
			comment := strings.TrimSpace(artifact.Evaluation.Comment)
			if comment == "" {
				comment = "(no comment)"
			}

			total += score
			histogram[score]++
			feedbackLines = append(feedbackLines, fmt.Sprintf("- [%d/10] %s", score, comment))

			if score < 5 {
				lowLines = append(lowLines, "- "+comment)
			}
			if score >= 8 {
				highLines = append(highLines, "- "+comment)
			}
		}

		avgScore = fmt.Sprintf("%.2f", float64(total)/float64(len(evaluated)))
		scoreHistogram = formatScoreHistogram(histogram)
		feedbackList = strings.Join(feedbackLines, "\n")

		if len(lowLines) > 0 {
			lowPatterns = strings.Join(lowLines, "\n")
		}
		if len(highLines) > 0 {
			highPatterns = strings.Join(highLines, "\n")
		}
	}

	directiveText := formatDirectives(directives)

	return fmt.Sprintf(`You are a master AI agent trainer. Improve the following agent based on evaluation feedback.

CURRENT AGENT (version %d):
System Prompt: %s

EVALUATION SUMMARY:
- Total artifacts: %d
- Evaluated artifacts: %d
- Average score: %s/10
- Score distribution: %s

FEEDBACK:
%s

LOW-SCORING PATTERNS (score < 5):
%s

HIGH-SCORING PATTERNS (score >= 8):
%s

DIRECTIVES:
%s

Output a JSON object with the following structure:
{
  "system_prompt": "the improved system prompt",
  "reasoning": "brief explanation of changes made"
}

Focus on addressing low-scoring feedback while preserving high-scoring behaviors.`,
		currentVersion,
		currentSystemPrompt,
		totalArtifacts,
		len(evaluated),
		avgScore,
		scoreHistogram,
		feedbackList,
		lowPatterns,
		highPatterns,
		directiveText,
	)
}

func latestAgentPrompt(agents []state.Agent) (int, string) {
	if len(agents) == 0 {
		return 0, "(none)"
	}

	latest := agents[0]
	for _, candidate := range agents[1:] {
		if candidate.Version > latest.Version {
			latest = candidate
		}
	}

	prompt := strings.TrimSpace(latest.Definition.SystemPrompt)
	if prompt == "" {
		prompt = "(none)"
	}

	return latest.Version, prompt
}

func evaluatedArtifacts(artifacts []state.Artifact) []state.Artifact {
	out := make([]state.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.Evaluation != nil {
			out = append(out, artifact)
		}
	}
	return out
}

func formatScoreHistogram(histogram map[int]int) string {
	if len(histogram) == 0 {
		return "No evaluation yet"
	}

	scores := make([]int, 0, len(histogram))
	for score := range histogram {
		scores = append(scores, score)
	}
	sort.Ints(scores)

	parts := make([]string, 0, len(scores))
	for _, score := range scores {
		parts = append(parts, fmt.Sprintf("%d:%d", score, histogram[score]))
	}

	return strings.Join(parts, ", ")
}

func formatDirectives(directives []state.Directive) string {
	if len(directives) == 0 {
		return "(none)"
	}

	lines := make([]string, 0, len(directives))
	for _, directive := range directives {
		text := strings.TrimSpace(directive.Text)
		if text == "" {
			continue
		}
		lines = append(lines, "- "+text)
	}

	if len(lines) == 0 {
		return "(none)"
	}

	return strings.Join(lines, "\n")
}
