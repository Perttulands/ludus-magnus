package cmd

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Perttulands/chiron/internal/sandbox"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
)

const (
	defaultSandboxModel        = "qwen3.5:9b-full"
	defaultSandboxProvider     = "ollama"
	defaultSandboxTimeout      = 10 * time.Minute
	defaultSandboxSystemPrompt = "You are running in a Chiron quarantine sandbox. Complete the user task using only the allowed tools."
)

type sandboxRunner interface {
	Run(
		ctx context.Context,
		cfg sandbox.Config,
		model, provider, systemPrompt, userPrompt, scenarioDir string,
	) (*sandbox.RunResult, error)
}

var (
	sandboxRunnerFactory = func() sandboxRunner { return &sandbox.Executor{} }
	sandboxNow           = func() time.Time { return time.Now().UTC() }
	sandboxGenerateRunID = func() string { return ulid.Make().String() }
)

type sandboxRunResultDoc struct {
	Outcome       string   `json:"outcome"`
	Turns         int      `json:"turns"`
	ToolCalls     int      `json:"tool_calls"`
	ToolCallNames []string `json:"tool_call_names,omitempty"`
	Tokens        struct {
		Input  int `json:"input"`
		Output int `json:"output"`
		Total  int `json:"total"`
	} `json:"tokens"`
	ExitCode  int `json:"exit_code"`
	EditCount int `json:"edit_count"`
}

type sandboxRunMetaDoc struct {
	RunID          string         `json:"run_id"`
	Timestamp      string         `json:"timestamp"`
	CompletedAt    string         `json:"completed_at"`
	DurationMS     int64          `json:"duration_ms"`
	Model          string         `json:"model"`
	Provider       string         `json:"provider"`
	SandboxEngine  string         `json:"sandbox_engine"`
	Offline        bool           `json:"offline"`
	Timeout        string         `json:"timeout"`
	Scenario       string         `json:"scenario"`
	Task           string         `json:"task"`
	TaskSummary    string         `json:"task_summary"`
	Turns          int            `json:"turns"`
	TokensIn       int            `json:"tokens_in"`
	TokensOut      int            `json:"tokens_out"`
	ToolCalls      int            `json:"tool_calls"`
	ExitCode       int            `json:"exit_code"`
	IsolationFlags map[string]any `json:"isolation"`
}

type sandboxReviewDoc struct {
	Approved   bool   `json:"approved"`
	ReviewedBy string `json:"reviewed_by"`
	TS         string `json:"ts"`
	Notes      string `json:"notes"`
}

type sandboxStatusRow struct {
	RunID       string
	Timestamp   time.Time
	TimestampS  string
	Model       string
	Outcome     string
	ReviewState string
	TaskSummary string
}

func newSandboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sandbox",
		Short: "Run, review, and inspect quarantine sandbox runs",
	}

	cmd.AddCommand(newSandboxRunCmd())
	cmd.AddCommand(newSandboxReviewCmd())
	cmd.AddCommand(newSandboxStatusCmd())

	return cmd
}

func newSandboxRunCmd() *cobra.Command {
	var (
		taskFile string
		model    string
		provider string
		offline  bool
		scenario string
		timeout  time.Duration
	)

	cmd := &cobra.Command{
		Use:   "run <task>",
		Short: "Run a task in the bwrap+Pi+Ollama sandbox",
		Args: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(taskFile) == "" && len(args) == 0 {
				return fmt.Errorf("task is required unless --file is set")
			}
			if strings.TrimSpace(taskFile) != "" && len(args) > 0 {
				return fmt.Errorf("provide task via arguments or --file, not both")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := loadSandboxTask(strings.Join(args, " "), taskFile)
			if err != nil {
				return err
			}

			scenarioDir, cleanupScenario, err := prepareSandboxScenario(scenario, task)
			if err != nil {
				return err
			}
			defer cleanupScenario()

			runsDir, err := quarantineRunsDir()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(runsDir, 0o755); err != nil {
				return fmt.Errorf("create runs dir: %w", err)
			}

			runID := sandboxGenerateRunID()
			runDir := filepath.Join(runsDir, runID)
			if err := os.MkdirAll(runDir, 0o755); err != nil {
				return fmt.Errorf("create run dir: %w", err)
			}

			runner := sandboxRunnerFactory()
			runCfg := sandbox.Config{
				Engine:  "bwrap",
				Tools:   []string{"read", "bash", "write"},
				Timeout: timeout,
			}

			start := sandboxNow()
			ctx := cmd.Context()
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			runFn := func() (*sandbox.RunResult, error) {
				return runner.Run(
					ctx,
					runCfg,
					model,
					provider,
					defaultSandboxSystemPrompt,
					task,
					scenarioDir,
				)
			}

			var result *sandbox.RunResult
			if offline {
				result, err = runWithOfflineBwrap(runFn)
			} else {
				result, err = runFn()
			}
			if err != nil {
				return fmt.Errorf("sandbox run failed: %w", err)
			}

			finish := sandboxNow()
			wallDuration := finish.Sub(start)

			if err := os.WriteFile(filepath.Join(runDir, "raw-output.jsonl"), result.RawOutput, 0o644); err != nil {
				return fmt.Errorf("write raw-output.jsonl: %w", err)
			}
			if err := os.WriteFile(filepath.Join(runDir, "workspace.diff"), []byte(result.WorkspaceDiff), 0o644); err != nil {
				return fmt.Errorf("write workspace.diff: %w", err)
			}
			workspaceArchive := filepath.Join(runDir, "workspace-after.tgz")
			if err := writeWorkspaceAfterArchive(scenarioDir, result.WorkspaceDiff, workspaceArchive); err != nil {
				return fmt.Errorf("write workspace-after.tgz: %w", err)
			}

			resultDoc := sandboxRunResultDoc{
				Outcome:       inferSandboxOutcome(result.RawOutput, result.ExitCode),
				Turns:         result.Turns,
				ToolCalls:     len(result.ToolCalls),
				ToolCallNames: result.ToolCalls,
				ExitCode:      result.ExitCode,
				EditCount:     result.EditCount,
			}
			resultDoc.Tokens.Input = result.TokensIn
			resultDoc.Tokens.Output = result.TokensOut
			resultDoc.Tokens.Total = result.TokensIn + result.TokensOut

			metaDoc := sandboxRunMetaDoc{
				RunID:         runID,
				Timestamp:     start.UTC().Format(time.RFC3339),
				CompletedAt:   finish.UTC().Format(time.RFC3339),
				DurationMS:    wallDuration.Milliseconds(),
				Model:         model,
				Provider:      provider,
				SandboxEngine: "bwrap",
				Offline:       offline,
				Timeout:       timeout.String(),
				Scenario:      scenarioDir,
				Task:          task,
				TaskSummary:   summarizeSandboxTask(task),
				Turns:         result.Turns,
				TokensIn:      result.TokensIn,
				TokensOut:     result.TokensOut,
				ToolCalls:     len(result.ToolCalls),
				ExitCode:      result.ExitCode,
				IsolationFlags: map[string]any{
					"clearenv":        true,
					"unshare_user":    true,
					"unshare_pid":     true,
					"unshare_uts":     true,
					"unshare_cgroup":  true,
					"die_with_parent": true,
					"unshare_net":     offline,
				},
			}

			if err := writeSandboxJSON(filepath.Join(runDir, "result.json"), resultDoc); err != nil {
				return fmt.Errorf("write result.json: %w", err)
			}
			if err := writeSandboxJSON(filepath.Join(runDir, "meta.json"), metaDoc); err != nil {
				return fmt.Errorf("write meta.json: %w", err)
			}

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{"run_id": runID})
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", runID)
			if err != nil {
				return fmt.Errorf("write output: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&taskFile, "file", "", "Read task prompt from file")
	cmd.Flags().StringVar(&model, "model", defaultSandboxModel, "Model to use")
	cmd.Flags().StringVar(&provider, "provider", defaultSandboxProvider, "Provider to use")
	cmd.Flags().BoolVar(&offline, "offline", false, "Air-gap the sandbox by unsharing network (--unshare-net)")
	cmd.Flags().StringVar(&scenario, "scenario", "", "Scenario directory to run against (default: temporary task-only scenario)")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultSandboxTimeout, "Wall-clock timeout")

	return cmd
}

func newSandboxReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review <run-id>",
		Short: "Review a sandbox run and approve/reject it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID := strings.TrimSpace(args[0])
			if runID == "" {
				return fmt.Errorf("run-id is required")
			}

			runsDir, err := quarantineRunsDir()
			if err != nil {
				return err
			}
			runDir := filepath.Join(runsDir, runID)

			metaPath := filepath.Join(runDir, "meta.json")
			resultPath := filepath.Join(runDir, "result.json")
			diffPath := filepath.Join(runDir, "workspace.diff")

			var meta sandboxRunMetaDoc
			_ = readSandboxJSON(metaPath, &meta)

			var result sandboxRunResultDoc
			_ = readSandboxJSON(resultPath, &result)

			diffBytes, _ := os.ReadFile(diffPath)
			diffText := string(diffBytes)

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "run-id: %s\n", runID)
			fmt.Fprintf(out, "timestamp: %s\n", meta.Timestamp)
			fmt.Fprintf(out, "model: %s\n", meta.Model)
			fmt.Fprintf(out, "provider: %s\n", meta.Provider)
			fmt.Fprintf(out, "outcome: %s\n", result.Outcome)
			fmt.Fprintf(out, "turns: %d\n", result.Turns)
			fmt.Fprintf(out, "tool_calls: %d\n", result.ToolCalls)
			fmt.Fprintln(out, "\nworkspace.diff:")
			if strings.TrimSpace(diffText) == "" {
				fmt.Fprintln(out, "(empty)")
			} else {
				fmt.Fprintln(out, diffText)
			}

			fmt.Fprint(out, "Approve this output entering Inside? [y/N] <notes> ")
			reader := bufio.NewReader(cmd.InOrStdin())
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return fmt.Errorf("read review input: %w", err)
			}
			line = strings.TrimSpace(line)

			approved, notes := parseSandboxReviewLine(line)
			review := sandboxReviewDoc{
				Approved:   approved,
				ReviewedBy: "human",
				TS:         sandboxNow().UTC().Format(time.RFC3339),
				Notes:      notes,
			}
			if err := writeSandboxJSON(filepath.Join(runDir, "review.json"), review); err != nil {
				return fmt.Errorf("write review.json: %w", err)
			}

			if approved {
				fmt.Fprintln(out, "review: approved")
			} else {
				fmt.Fprintln(out, "review: rejected")
			}
			return nil
		},
	}

	return cmd
}

func newSandboxStatusCmd() *cobra.Command {
	var showAll bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "List quarantine sandbox runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			runsDir, err := quarantineRunsDir()
			if err != nil {
				return err
			}
			rows, err := loadSandboxStatusRows(runsDir)
			if err != nil {
				return err
			}

			if len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No sandbox runs found.")
				return nil
			}

			sort.Slice(rows, func(i, j int) bool {
				return rows[i].Timestamp.After(rows[j].Timestamp)
			})

			if !showAll && len(rows) > 10 {
				rows = rows[:10]
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%-26s %-20s %-22s %-10s %-10s %s\n", "run-id", "ts", "model", "outcome", "review", "task")
			for _, row := range rows {
				fmt.Fprintf(out, "%-26s %-20s %-22s %-10s %-10s %s\n",
					row.RunID,
					row.TimestampS,
					truncateSandboxField(row.Model, 22),
					truncateSandboxField(row.Outcome, 10),
					truncateSandboxField(row.ReviewState, 10),
					row.TaskSummary,
				)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "Show all runs (default: last 10)")
	return cmd
}

func loadSandboxTask(argTask, taskFile string) (string, error) {
	if strings.TrimSpace(taskFile) != "" {
		data, err := os.ReadFile(taskFile)
		if err != nil {
			return "", fmt.Errorf("read task file: %w", err)
		}
		task := strings.TrimSpace(string(data))
		if task == "" {
			return "", fmt.Errorf("task file is empty")
		}
		return task, nil
	}

	task := strings.TrimSpace(argTask)
	if task == "" {
		return "", fmt.Errorf("task is empty")
	}
	return task, nil
}

func prepareSandboxScenario(scenarioFlag, task string) (scenarioDir string, cleanup func(), err error) {
	scenarioFlag = strings.TrimSpace(scenarioFlag)
	if scenarioFlag != "" {
		absScenario, err := filepath.Abs(scenarioFlag)
		if err != nil {
			return "", nil, fmt.Errorf("resolve scenario path: %w", err)
		}
		info, err := os.Stat(absScenario)
		if err != nil {
			return "", nil, fmt.Errorf("stat scenario: %w", err)
		}
		if !info.IsDir() {
			return "", nil, fmt.Errorf("scenario path is not a directory: %s", absScenario)
		}
		return absScenario, func() {}, nil
	}

	tmpScenario, err := os.MkdirTemp("", "chiron-sandbox-scenario-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp scenario: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmpScenario, "task.txt"), []byte(task+"\n"), 0o644); err != nil {
		os.RemoveAll(tmpScenario)
		return "", nil, fmt.Errorf("write task.txt: %w", err)
	}
	return tmpScenario, func() { _ = os.RemoveAll(tmpScenario) }, nil
}

func quarantineRunsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".quarantine", "runs"), nil
}

func runWithOfflineBwrap(runFn func() (*sandbox.RunResult, error)) (*sandbox.RunResult, error) {
	realBwrap, err := exec.LookPath("bwrap")
	if err != nil {
		return nil, fmt.Errorf("offline mode requested but bwrap is unavailable: %w", err)
	}

	wrapperDir, err := os.MkdirTemp("", "chiron-offline-bwrap-*")
	if err != nil {
		return nil, fmt.Errorf("create bwrap wrapper dir: %w", err)
	}
	defer os.RemoveAll(wrapperDir)

	wrapperPath := filepath.Join(wrapperDir, "bwrap")
	script := fmt.Sprintf("#!/usr/bin/env bash\nset -euo pipefail\nexec %q --unshare-net \"$@\"\n", realBwrap)
	if err := os.WriteFile(wrapperPath, []byte(script), 0o755); err != nil {
		return nil, fmt.Errorf("write bwrap wrapper: %w", err)
	}

	oldPath := os.Getenv("PATH")
	newPath := wrapperDir
	if oldPath != "" {
		newPath = wrapperDir + string(os.PathListSeparator) + oldPath
	}
	if err := os.Setenv("PATH", newPath); err != nil {
		return nil, fmt.Errorf("set PATH for offline bwrap wrapper: %w", err)
	}
	defer func() {
		_ = os.Setenv("PATH", oldPath)
	}()

	return runFn()
}

func writeWorkspaceAfterArchive(scenarioDir, workspaceDiff, archivePath string) error {
	replayRoot, err := os.MkdirTemp("", "chiron-workspace-replay-*")
	if err != nil {
		return fmt.Errorf("create replay workspace: %w", err)
	}
	defer os.RemoveAll(replayRoot)

	replayWorkspace := filepath.Join(replayRoot, "workspace")
	if err := os.MkdirAll(replayWorkspace, 0o755); err != nil {
		return fmt.Errorf("create replay workspace dir: %w", err)
	}
	if err := copyDirContents(scenarioDir, replayWorkspace); err != nil {
		return fmt.Errorf("copy scenario into replay workspace: %w", err)
	}

	if strings.TrimSpace(workspaceDiff) != "" {
		normalized := normalizeWorkspaceDiff(workspaceDiff, scenarioDir)
		if strings.TrimSpace(normalized) != "" {
			_ = applyPatchToWorkspace(replayWorkspace, normalized)
		}
	}

	return writeTarGzFromDir(replayWorkspace, archivePath)
}

func normalizeWorkspaceDiff(diffText, scenarioDir string) string {
	scenarioDir = filepath.Clean(scenarioDir)
	lines := strings.SplitAfter(diffText, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "--- ") {
			lines[i] = rewritePatchHeader(line, "--- ", scenarioDir)
			continue
		}
		if strings.HasPrefix(line, "+++ ") {
			lines[i] = rewritePatchHeader(line, "+++ ", scenarioDir)
		}
	}
	return strings.Join(lines, "")
}

func rewritePatchHeader(line, marker, scenarioDir string) string {
	lineNoMarker := strings.TrimPrefix(line, marker)
	path, suffix := splitPatchHeaderPath(lineNoMarker)
	if path == "" || path == "/dev/null" {
		return marker + path + suffix
	}

	rel, ok := relativePatchPath(path, scenarioDir)
	if !ok {
		return line
	}

	if rel == "." {
		rel = ""
	}
	prefix := "a/"
	if marker == "+++ " {
		prefix = "b/"
	}
	return marker + prefix + rel + suffix
}

func splitPatchHeaderPath(s string) (path, suffix string) {
	idx := strings.Index(s, "\t")
	if idx < 0 {
		trimmed := strings.TrimRight(s, "\n")
		return trimmed, s[len(trimmed):]
	}
	return s[:idx], s[idx:]
}

func relativePatchPath(path, scenarioDir string) (string, bool) {
	clean := filepath.Clean(path)
	if clean == scenarioDir {
		return ".", true
	}
	if strings.HasPrefix(clean, scenarioDir+string(os.PathSeparator)) {
		return strings.TrimPrefix(clean, scenarioDir+string(os.PathSeparator)), true
	}

	workspaceMarker := string(os.PathSeparator) + "workspace" + string(os.PathSeparator)
	if idx := strings.Index(clean, workspaceMarker); idx >= 0 {
		return clean[idx+len(workspaceMarker):], true
	}
	if strings.HasSuffix(clean, string(os.PathSeparator)+"workspace") {
		return ".", true
	}

	return "", false
}

func applyPatchToWorkspace(workspaceDir, patchText string) error {
	patchPath := filepath.Join(workspaceDir, ".chiron-workspace.patch")
	if err := os.WriteFile(patchPath, []byte(patchText), 0o644); err != nil {
		return err
	}
	defer os.Remove(patchPath)

	cmd := exec.Command("patch", "-p1", "-i", patchPath)
	cmd.Dir = workspaceDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("patch failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func writeTarGzFromDir(sourceDir, archivePath string) error {
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer archiveFile.Close()

	gzWriter := gzip.NewWriter(archiveFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	excluded := map[string]bool{
		".lab-bin":    true,
		".lab-br.log": true,
	}

	return filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if excluded[d.Name()] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		var linkTarget string
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}

		header, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		_, err = io.Copy(tarWriter, in)
		return err
	})
}

func copyDirContents(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		if err := copyPath(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyPath(srcPath, dstPath string) error {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(srcPath)
		if err != nil {
			return err
		}
		return os.Symlink(target, dstPath)
	}

	if info.IsDir() {
		if err := os.MkdirAll(dstPath, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(srcPath)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyPath(filepath.Join(srcPath, entry.Name()), filepath.Join(dstPath, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func inferSandboxOutcome(rawOutput []byte, exitCode int) string {
	scanner := bufio.NewScanner(bytes.NewReader(rawOutput))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		t, _ := obj["type"].(string)
		if t == "result" {
			if subtype, ok := obj["subtype"].(string); ok && strings.TrimSpace(subtype) != "" {
				return subtype
			}
		}
		if t == "agent_end" {
			if subtype, ok := obj["subtype"].(string); ok && strings.TrimSpace(subtype) != "" {
				return subtype
			}
		}
	}

	if exitCode == 0 {
		return "success"
	}
	return "error"
}

func summarizeSandboxTask(task string) string {
	summary := strings.Join(strings.Fields(task), " ")
	if summary == "" {
		return "(empty task)"
	}
	if len(summary) <= 80 {
		return summary
	}
	return summary[:77] + "..."
}

func writeSandboxJSON(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readSandboxJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func parseSandboxReviewLine(line string) (approved bool, notes string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return false, ""
	}
	parts := strings.Fields(line)
	first := strings.ToLower(parts[0])
	switch first {
	case "y", "yes":
		notes = strings.TrimSpace(strings.TrimPrefix(line, parts[0]))
		return true, notes
	default:
		return false, line
	}
}

func loadSandboxStatusRows(runsDir string) ([]sandboxStatusRow, error) {
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read runs dir: %w", err)
	}

	rows := make([]sandboxStatusRow, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runID := entry.Name()
		runDir := filepath.Join(runsDir, runID)

		metaPath := filepath.Join(runDir, "meta.json")
		resultPath := filepath.Join(runDir, "result.json")
		reviewPath := filepath.Join(runDir, "review.json")

		meta := sandboxRunMetaDoc{}
		_ = readSandboxJSON(metaPath, &meta)

		result := sandboxRunResultDoc{}
		_ = readSandboxJSON(resultPath, &result)

		reviewState := "pending"
		review := sandboxReviewDoc{}
		if err := readSandboxJSON(reviewPath, &review); err == nil {
			if review.Approved {
				reviewState = "approved"
			} else {
				reviewState = "rejected"
			}
		}

		timestamp := parseSandboxTimestamp(meta.Timestamp)
		if timestamp.IsZero() {
			if info, err := os.Stat(runDir); err == nil {
				timestamp = info.ModTime().UTC()
			}
		}

		tsString := "unknown"
		if !timestamp.IsZero() {
			tsString = timestamp.Format(time.RFC3339)
		}

		taskSummary := strings.TrimSpace(meta.TaskSummary)
		if taskSummary == "" {
			taskSummary = summarizeSandboxTask(meta.Task)
		}

		outcome := result.Outcome
		if strings.TrimSpace(outcome) == "" {
			outcome = "unknown"
		}

		rows = append(rows, sandboxStatusRow{
			RunID:       runID,
			Timestamp:   timestamp,
			TimestampS:  tsString,
			Model:       meta.Model,
			Outcome:     outcome,
			ReviewState: reviewState,
			TaskSummary: taskSummary,
		})
	}

	return rows, nil
}

func parseSandboxTimestamp(ts string) time.Time {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func truncateSandboxField(value string, max int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}
