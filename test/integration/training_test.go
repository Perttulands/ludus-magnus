package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type trainingOpenAIRequest struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func TestTrainingFlowWithPromotionAndLocks(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}

		var req trainingOpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user := ""
		system := ""
		for _, msg := range req.Messages {
			switch msg.Role {
			case "user":
				user = msg.Content
			case "system":
				system = msg.Content
			}
		}

		responseText := "default-response"
		switch {
		case strings.Contains(user, "Promotion strategy:") && strings.Contains(user, "conservative approach"):
			responseText = "lineage-A-base"
		case strings.Contains(user, "Promotion strategy:") && strings.Contains(user, "balanced approach"):
			responseText = "lineage-B-base"
		case strings.Contains(user, "Promotion strategy:") && strings.Contains(user, "creative approach"):
			responseText = "lineage-C-base"
		case strings.Contains(user, "Promotion strategy:") && strings.Contains(user, "aggressive approach"):
			responseText = "lineage-D-base"
		case strings.Contains(user, "Improve the following agent based on evaluation feedback") && strings.Contains(user, "lineage-A-base"):
			responseText = "lineage-A-evolved-v2"
		case strings.Contains(user, "Improve the following agent based on evaluation feedback") && strings.Contains(user, "lineage-C-base"):
			responseText = "lineage-C-evolved-v2"
		case strings.Contains(system, "lineage-A-evolved-v2"):
			responseText = "run-output-A-v2"
		case strings.Contains(system, "lineage-C-evolved-v2"):
			responseText = "run-output-C-v2"
		case strings.Contains(system, "lineage-A-base"):
			responseText = "run-output-A-v1"
		case strings.Contains(system, "lineage-B-base"):
			responseText = "run-output-B-v1"
		case strings.Contains(system, "lineage-C-base"):
			responseText = "run-output-C-v1"
		case strings.Contains(system, "lineage-D-base"):
			responseText = "run-output-D-v1"
		case strings.Contains(user, "master AI agent trainer"):
			responseText = "quickstart-main-v1"
		}

		payload := map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": responseText}}},
			"usage": map[string]any{"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	repoRoot := findRepoRoot(t)
	workspace := mustMkdirTemp(t, "training-int-*")
	defer func() {
		_ = os.RemoveAll(workspace)
		statePath := filepath.Join(workspace, ".ludus-magnus", "state.json")
		if _, err := os.Stat(statePath); !os.IsNotExist(err) {
			t.Errorf("expected temporary state path to be cleaned up: %s", statePath)
		}
	}()

	binaryPath := filepath.Join(workspace, "ludus-magnus")
	runCmd(t, repoRoot, "/usr/local/go/bin/go", "build", "-o", binaryPath, ".")

	initOut := runCmd(t, workspace, binaryPath,
		"quickstart", "init",
		"--need", "support workflow assistant",
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	)
	sessionID := captureMatch(t, `session_id=(ses_[a-f0-9]{8})`, initOut)

	promoteOut := runCmd(t, workspace, binaryPath,
		"promote", sessionID,
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	)
	if !strings.Contains(promoteOut, "Session promoted to training mode with 4 lineages") {
		t.Fatalf("unexpected promote output: %s", promoteOut)
	}

	artA := captureMatch(t, `artifact_id=(art_[a-f0-9]{8})`, runCmd(t, workspace, binaryPath,
		"run", sessionID,
		"--lineage", "A",
		"--input", "answer customer request",
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	))
	artB := captureMatch(t, `artifact_id=(art_[a-f0-9]{8})`, runCmd(t, workspace, binaryPath,
		"run", sessionID,
		"--lineage", "B",
		"--input", "answer customer request",
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	))
	artC := captureMatch(t, `artifact_id=(art_[a-f0-9]{8})`, runCmd(t, workspace, binaryPath,
		"run", sessionID,
		"--lineage", "C",
		"--input", "answer customer request",
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	))
	artD := captureMatch(t, `artifact_id=(art_[a-f0-9]{8})`, runCmd(t, workspace, binaryPath,
		"run", sessionID,
		"--lineage", "D",
		"--input", "answer customer request",
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	))

	runCmd(t, workspace, binaryPath, "evaluate", artA, "--score", "2", "--comment", "too verbose")
	runCmd(t, workspace, binaryPath, "evaluate", artB, "--score", "9", "--comment", "high quality")
	runCmd(t, workspace, binaryPath, "evaluate", artC, "--score", "3", "--comment", "incomplete")
	runCmd(t, workspace, binaryPath, "evaluate", artD, "--score", "8", "--comment", "useful")

	runCmd(t, workspace, binaryPath, "lineage", "lock", sessionID, "B")
	runCmd(t, workspace, binaryPath, "lineage", "lock", sessionID, "D")

	iterateOut := runCmd(t, workspace, binaryPath,
		"training", "iterate", sessionID,
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	)
	if !strings.Contains(iterateOut, "Regenerated 2 lineages: A, C. Locked: B, D.") {
		t.Fatalf("unexpected training iterate output: %s", iterateOut)
	}

	statePath := filepath.Join(workspace, ".ludus-magnus", "state.json")
	rawState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}

	var stateDoc struct {
		Sessions map[string]struct {
			Mode     string `json:"mode"`
			Lineages map[string]struct {
				Name   string `json:"name"`
				Locked bool   `json:"locked"`
				Agents []struct {
					Version    int `json:"version"`
					Definition struct {
						SystemPrompt string `json:"system_prompt"`
					} `json:"definition"`
				} `json:"agents"`
			} `json:"lineages"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(rawState, &stateDoc); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}

	session, ok := stateDoc.Sessions[sessionID]
	if !ok {
		t.Fatalf("session not found in state: %s", sessionID)
	}
	if session.Mode != "training" {
		t.Fatalf("expected session mode training after promote, got %q", session.Mode)
	}
	if len(session.Lineages) != 4 {
		t.Fatalf("expected 4 lineages after promote, got %d", len(session.Lineages))
	}

	for _, name := range []string{"B", "D"} {
		lineage, ok := findLineageFromDecodedSession(session, name)
		if !ok {
			t.Fatalf("expected lineage %s to exist", name)
		}
		if !lineage.Locked {
			t.Fatalf("expected lineage %s to remain locked", name)
		}
		if len(lineage.Agents) != 1 {
			t.Fatalf("expected locked lineage %s to keep 1 agent, got %d", name, len(lineage.Agents))
		}
	}

	for _, name := range []string{"A", "C"} {
		lineage, ok := findLineageFromDecodedSession(session, name)
		if !ok {
			t.Fatalf("expected lineage %s to exist", name)
		}
		if lineage.Locked {
			t.Fatalf("expected lineage %s to remain unlocked", name)
		}
		if len(lineage.Agents) != 2 {
			t.Fatalf("expected unlocked lineage %s to have 2 agents, got %d", name, len(lineage.Agents))
		}
		if lineage.Agents[1].Version != 2 {
			t.Fatalf("expected unlocked lineage %s new version=2, got %d", name, lineage.Agents[1].Version)
		}
		if !strings.Contains(lineage.Agents[1].Definition.SystemPrompt, "evolved-v2") {
			t.Fatalf("expected unlocked lineage %s to evolve prompt, got %q", name, lineage.Agents[1].Definition.SystemPrompt)
		}
	}
}

func findLineageFromDecodedSession(session struct {
	Mode     string `json:"mode"`
	Lineages map[string]struct {
		Name   string `json:"name"`
		Locked bool   `json:"locked"`
		Agents []struct {
			Version    int `json:"version"`
			Definition struct {
				SystemPrompt string `json:"system_prompt"`
			} `json:"definition"`
		} `json:"agents"`
	} `json:"lineages"`
}, name string) (struct {
	Name   string `json:"name"`
	Locked bool   `json:"locked"`
	Agents []struct {
		Version    int `json:"version"`
		Definition struct {
			SystemPrompt string `json:"system_prompt"`
		} `json:"definition"`
	} `json:"agents"`
}, bool) {
	for _, lineage := range session.Lineages {
		if lineage.Name == name {
			return lineage, true
		}
	}
	return struct {
		Name   string `json:"name"`
		Locked bool   `json:"locked"`
		Agents []struct {
			Version    int `json:"version"`
			Definition struct {
				SystemPrompt string `json:"system_prompt"`
			} `json:"definition"`
		} `json:"agents"`
	}{}, false
}
