package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type openAIRequest struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func TestQuickstartFlowEndToEnd(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}

		var req openAIRequest
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
		case strings.Contains(user, "Improve the following agent based on evaluation feedback"):
			responseText = "Evolved system prompt v2"
		case strings.Contains(user, "master AI agent trainer"):
			responseText = "Baseline system prompt v1"
		case strings.Contains(system, "Evolved system prompt v2"):
			responseText = "execution-output-v2"
		default:
			responseText = "execution-output-v1"
		}

		payload := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": responseText,
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     12,
				"completion_tokens": 8,
				"total_tokens":      20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	repoRoot := findRepoRoot(t)
	workspace := mustMkdirTemp(t, "quickstart-int-*")
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
		"--need", "support agent",
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	)

	sessionID := captureMatch(t, `session_id=(ses_[a-f0-9]{8})`, initOut)
	artifactV1 := captureMatch(t, `artifact_id=(art_[a-f0-9]{8})`, runCmd(t, workspace, binaryPath,
		"run", sessionID,
		"--input", "How do I reset my password?",
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	))

	runCmd(t, workspace, binaryPath, "evaluate", artifactV1, "--score", "3", "--comment", "too generic")

	iterateOut := runCmd(t, workspace, binaryPath,
		"iterate", sessionID,
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	)
	if !strings.Contains(iterateOut, "version=2") {
		t.Fatalf("expected iterate output to contain version=2, got:\n%s", iterateOut)
	}

	_ = captureMatch(t, `artifact_id=(art_[a-f0-9]{8})`, runCmd(t, workspace, binaryPath,
		"run", sessionID,
		"--input", "How do I reset my password?",
		"--provider", "openai-compatible",
		"--api-key", "test-key",
		"--base-url", server.URL,
	))

	statePath := filepath.Join(workspace, ".ludus-magnus", "state.json")
	rawState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}

	var stateDoc struct {
		Sessions map[string]struct {
			Lineages map[string]struct {
				Agents []struct {
					Version int `json:"version"`
				} `json:"agents"`
				Artifacts []struct {
					Output string `json:"output"`
				} `json:"artifacts"`
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

	var lineageKey string
	for k := range session.Lineages {
		lineageKey = k
		break
	}
	if lineageKey == "" {
		t.Fatalf("expected at least one lineage in session %s", sessionID)
	}

	lineage := session.Lineages[lineageKey]
	if len(lineage.Agents) < 2 {
		t.Fatalf("expected at least 2 agents after iterate, got %d", len(lineage.Agents))
	}
	if len(lineage.Artifacts) < 2 {
		t.Fatalf("expected at least 2 artifacts after two runs, got %d", len(lineage.Artifacts))
	}
	if lineage.Artifacts[0].Output == lineage.Artifacts[1].Output {
		t.Fatalf("expected v2 output to differ from v1 output, both were %q", lineage.Artifacts[0].Output)
	}
}

func runCmd(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH=/usr/local/go/bin:"+os.Getenv("PATH"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %s %s\nstdout:\n%s\nstderr:\n%s\nerror: %v",
			name, strings.Join(args, " "), stdout.String(), stderr.String(), err)
	}

	return stdout.String()
}

func captureMatch(t *testing.T, pattern string, text string) string {
	t.Helper()

	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(text)
	if len(m) != 2 {
		t.Fatalf("pattern %q not found in output:\n%s", pattern, text)
	}
	return m[1]
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod from %s", dir)
		}
		dir = parent
	}
}

func mustMkdirTemp(t *testing.T, pattern string) string {
	t.Helper()

	dir, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatalf("mkdirtemp: %v", err)
	}
	return dir
}
