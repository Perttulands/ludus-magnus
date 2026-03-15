package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config holds sandbox execution parameters.
type Config struct {
	Engine  string   // "bwrap" | "none"
	Tools   []string // Pi tool allowlist
	BrStub  bool     // inject br stub binary
	Timeout time.Duration
}

// RunResult captures the outcome of a sandboxed pi run.
type RunResult struct {
	RawOutput     []byte
	WorkspaceDiff string
	BrLog         string
	ExitCode      int
	DurationMs    int
	Turns         int
	ToolCalls     []string
	EditCount     int
	TokensIn      int
	TokensOut     int
}

// Executor runs pi inside a sandbox.
type Executor struct{}

// Run executes pi in the configured sandbox engine and returns results.
func (e *Executor) Run(
	ctx context.Context,
	cfg Config,
	model, provider, systemPrompt, userPrompt, scenarioDir string,
) (*RunResult, error) {

	// ------------------------------------------------------------------ //
	// 1. Create temp workspace
	// ------------------------------------------------------------------ //
	workParent, err := os.MkdirTemp("", "chiron-sandbox-*")
	if err != nil {
		return nil, fmt.Errorf("mktemp work_parent: %w", err)
	}
	defer os.RemoveAll(workParent)

	workspace := filepath.Join(workParent, "workspace")
	if err := copyDir(scenarioDir, workspace); err != nil {
		return nil, fmt.Errorf("copy scenario: %w", err)
	}

	// ------------------------------------------------------------------ //
	// 2. Fake HOME
	// ------------------------------------------------------------------ //
	fakeHome := filepath.Join(workParent, ".fake-home")
	piAgentDir := filepath.Join(fakeHome, ".pi", "agent")
	if err := os.MkdirAll(piAgentDir, 0o755); err != nil {
		return nil, fmt.Errorf("create fake home: %w", err)
	}
	// Copy models.json if available
	if home, ok := os.LookupEnv("HOME"); ok {
		src := filepath.Join(home, ".pi", "agent", "models.json")
		dst := filepath.Join(piAgentDir, "models.json")
		_ = copyFile(src, dst) // best-effort
	}

	// ------------------------------------------------------------------ //
	// 3. Scratch dirs
	// ------------------------------------------------------------------ //
	gopath := filepath.Join(workParent, ".gopath")
	gomodcache := filepath.Join(workParent, ".gomodcache")
	tmpDir := filepath.Join(workParent, ".tmp")
	cacheDir := filepath.Join(workParent, ".cache")
	for _, d := range []string{gopath, gomodcache, tmpDir, cacheDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("create scratch dir %s: %w", d, err)
		}
	}

	// ------------------------------------------------------------------ //
	// 4. Br stub
	// ------------------------------------------------------------------ //
	brLog := filepath.Join(workspace, ".lab-br.log")
	if cfg.BrStub {
		labBin := filepath.Join(workspace, ".lab-bin")
		if err := os.MkdirAll(labBin, 0o755); err != nil {
			return nil, fmt.Errorf("create .lab-bin: %w", err)
		}
		stubScript := `#!/usr/bin/env bash
set -euo pipefail
log_path="${BR_STUB_LOG:-./.lab-br.log}"
timestamp="$(date -Iseconds)"
printf '%s\t%s\n' "$timestamp" "$*" >> "$log_path"
echo "br-stub: logged invocation"
`
		stubPath := filepath.Join(labBin, "br")
		if err := os.WriteFile(stubPath, []byte(stubScript), 0o755); err != nil {
			return nil, fmt.Errorf("write br stub: %w", err)
		}
	}

	// ------------------------------------------------------------------ //
	// Resolve binary paths
	// ------------------------------------------------------------------ //
	piPath, err := exec.LookPath("pi")
	if err != nil {
		return nil, fmt.Errorf("pi not found in PATH: %w", err)
	}
	piDir := filepath.Dir(piPath)

	goExe, err := exec.LookPath("go")
	if err != nil {
		goExe = "/usr/local/go/bin/go"
	}
	goDir := filepath.Dir(goExe)
	goRoot := os.Getenv("GOROOT")
	if goRoot == "" {
		// Try to derive GOROOT from go binary location
		out, err2 := exec.Command(goExe, "env", "GOROOT").Output()
		if err2 == nil {
			goRoot = strings.TrimSpace(string(out))
		}
	}

	nodeExe, nodeErr := exec.LookPath("node")
	nodeDir := ""
	if nodeErr == nil {
		nodeDir = filepath.Dir(nodeExe)
	}

	// ------------------------------------------------------------------ //
	// 5/6. Build sandbox PATH
	// ------------------------------------------------------------------ //
	sandboxPath := buildSandboxPath(cfg.BrStub, workspace, piDir, goDir, nodeDir)

	// ------------------------------------------------------------------ //
	// 7. Pi command arguments
	// ------------------------------------------------------------------ //
	piArgs := buildPiArgs(provider, model, systemPrompt, cfg.Tools)

	// ------------------------------------------------------------------ //
	// 8/9. Execute
	// ------------------------------------------------------------------ //
	var cmd *exec.Cmd
	start := time.Now()

	switch cfg.Engine {
	case "bwrap":
		bwrapPath, err2 := exec.LookPath("bwrap")
		if err2 != nil {
			return nil, fmt.Errorf("bwrap not found: %w", err2)
		}
		bwrapArgs := BuildBwrapArgs(cfg, workspace, fakeHome, workParent, goRoot, sandboxPath, brLog)
		// Append the pi command
		bwrapArgs = append(bwrapArgs, piPath)
		bwrapArgs = append(bwrapArgs, piArgs...)
		cmd = exec.CommandContext(ctx, bwrapPath, bwrapArgs...) // ubs:ignore - intentional: bwrap sandbox, args are config-derived not user input

	case "none":
		cmd = exec.CommandContext(ctx, piPath, piArgs...)
		cmd.Env = buildDirectEnv(workspace, fakeHome, gopath, gomodcache, tmpDir, cacheDir, goRoot, sandboxPath)
		cmd.Dir = workspace

	default:
		return nil, fmt.Errorf("unknown sandbox engine %q", cfg.Engine)
	}

	cmd.Stdin = strings.NewReader(userPrompt)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr // let stderr flow through for debugging

	runErr := cmd.Run()
	durationMs := int(time.Since(start).Milliseconds())

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Non-exit error (e.g. context cancelled) — still try to collect output
			return nil, fmt.Errorf("run pi: %w", runErr)
		}
	}

	// ------------------------------------------------------------------ //
	// 10. Post-run collection
	// ------------------------------------------------------------------ //
	rawOutput := stdout.Bytes()

	// Workspace diff
	diffOut, _ := runDiff(scenarioDir, workspace)

	// Br log
	brLogContent := ""
	if data, err2 := os.ReadFile(brLog); err2 == nil {
		brLogContent = string(data)
	}

	// ------------------------------------------------------------------ //
	// 11. Parse JSONL
	// ------------------------------------------------------------------ //
	turns, toolCalls, editCount, tokensIn, tokensOut := ParseJSONL(rawOutput)

	return &RunResult{
		RawOutput:     rawOutput,
		WorkspaceDiff: diffOut,
		BrLog:         brLogContent,
		ExitCode:      exitCode,
		DurationMs:    durationMs,
		Turns:         turns,
		ToolCalls:     toolCalls,
		EditCount:     editCount,
		TokensIn:      tokensIn,
		TokensOut:     tokensOut,
	}, nil
}

// ------------------------------------------------------------------ //
// BuildBwrapArgs builds the bwrap argument list (exported for testing).
// ------------------------------------------------------------------ //
func BuildBwrapArgs(cfg Config, workspace, fakeHome, workParent, goRoot, sandboxPath, brLog string) []string {
	gopath := filepath.Join(workParent, ".gopath")
	gomodcache := filepath.Join(workParent, ".gomodcache")
	tmpDir := filepath.Join(workParent, ".tmp")
	cacheDir := filepath.Join(workParent, ".cache")

	piPath, _ := exec.LookPath("pi")
	piDir := filepath.Dir(piPath)
	nodeExe, nodeErr := exec.LookPath("node")
	nodeDir := ""
	if nodeErr == nil {
		nodeDir = filepath.Dir(nodeExe)
	}

	args := []string{
		"--clearenv",
		// Read-only OS binds
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/etc", "/etc",
		"--symlink", "usr/lib64", "/lib64",
		// Proc/dev/tmp
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		// Writable binds
		"--bind", workspace, workspace,
		"--bind", fakeHome, fakeHome,
		"--bind", gopath, gopath,
		"--bind", gomodcache, gomodcache,
		"--bind", tmpDir, tmpDir,
		"--bind", cacheDir, cacheDir,
	}

	// Go root (read-only)
	if goRoot != "" {
		args = append(args, "--ro-bind", goRoot, goRoot)
	}

	// Pi binary dir (ro-bind if outside /usr)
	if piDir != "" && !strings.HasPrefix(piDir, "/usr") {
		args = append(args, "--ro-bind", piDir, piDir)
	}

	// Node binary dir (ro-bind if outside /usr)
	if nodeDir != "" && !strings.HasPrefix(nodeDir, "/usr") {
		args = append(args, "--ro-bind", nodeDir, nodeDir)
		// node lib/node_modules
		nodeLibModules := filepath.Join(filepath.Dir(nodeDir), "lib", "node_modules")
		if _, err := os.Stat(nodeLibModules); err == nil {
			args = append(args, "--ro-bind", nodeLibModules, nodeLibModules)
		}
	}

	// Working directory
	args = append(args, "--chdir", workspace)

	// Isolation flags
	args = append(args,
		"--unshare-user",
		"--unshare-pid",
		"--unshare-uts",
		"--unshare-cgroup",
		"--die-with-parent",
	)

	// Environment
	goRoot2 := goRoot
	if goRoot2 == "" {
		goRoot2 = "/usr/local/go"
	}
	envVars := []struct{ k, v string }{
		{"HOME", fakeHome},
		{"PATH", sandboxPath},
		{"USER", "sandbox"},
		{"LANG", "C.UTF-8"},
		{"PI_OFFLINE", "1"},
		{"BR_STUB_LOG", brLog},
		{"GOPATH", gopath},
		{"GOMODCACHE", gomodcache},
		{"GOROOT", goRoot2},
		{"TMPDIR", tmpDir},
		{"XDG_CACHE_HOME", cacheDir},
		{"TERM", "dumb"},
		{"SHELL", "/bin/bash"},
	}
	for _, kv := range envVars {
		args = append(args, "--setenv", kv.k, kv.v)
	}

	return args
}

// ------------------------------------------------------------------ //
// ParseJSONL parses pi's JSONL output (exported for testing).
// ------------------------------------------------------------------ //
func ParseJSONL(data []byte) (turns int, toolCalls []string, editCount int, tokensIn int, tokensOut int) {
	toolCalls = []string{}
	scanner := bytes.NewReader(data)
	dec := json.NewDecoder(scanner)

	for {
		var raw map[string]json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			// Skip malformed lines by re-seeking is not straightforward with
			// json.Decoder; handle line-by-line instead.
			break
		}
		processJSONObject(raw, &turns, &toolCalls, &tokensIn, &tokensOut)
	}

	// Fall back to line-by-line if decoder ate everything or had errors
	// (json.Decoder handles concatenated JSON objects, but lines with parse
	// errors are silently dropped above — re-scan line-by-line for safety).
	if turns == 0 && len(toolCalls) == 0 {
		turns, toolCalls, tokensIn, tokensOut = parseJSONLLines(data)
	}

	editCount = countEdits(toolCalls)
	return
}

func parseJSONLLines(data []byte) (turns int, toolCalls []string, tokensIn int, tokensOut int) {
	toolCalls = []string{}
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		processJSONObject(raw, &turns, &toolCalls, &tokensIn, &tokensOut)
	}
	return
}

func processJSONObject(raw map[string]json.RawMessage, turns *int, toolCalls *[]string, tokensIn, tokensOut *int) {
	typeRaw, ok := raw["type"]
	if !ok {
		return
	}
	var typ string
	if err := json.Unmarshal(typeRaw, &typ); err != nil {
		return
	}

	switch typ {
	case "turn_end":
		*turns++
		// Extract message.usage.input and message.usage.output
		if msgRaw, ok := raw["message"]; ok {
			var msg struct {
				Usage struct {
					Input  int `json:"input"`
					Output int `json:"output"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(msgRaw, &msg); err == nil {
				*tokensIn += msg.Usage.Input
				*tokensOut += msg.Usage.Output
			}
		}

	case "tool_execution_start":
		if nameRaw, ok := raw["toolName"]; ok {
			var name string
			if err := json.Unmarshal(nameRaw, &name); err == nil {
				*toolCalls = append(*toolCalls, name)
			}
		}
	}
}

func countEdits(toolCalls []string) int {
	count := 0
	for _, tc := range toolCalls {
		lower := strings.ToLower(tc)
		if strings.Contains(lower, "edit") || strings.Contains(lower, "write") {
			count++
		}
	}
	return count
}

// ------------------------------------------------------------------ //
// Internal helpers
// ------------------------------------------------------------------ //

func buildPiArgs(provider, model, systemPrompt string, tools []string) []string {
	args := []string{
		"-p",
		"--provider", provider,
		"--model", model,
		"--system-prompt", systemPrompt,
		"--mode", "json",
		"--no-session",
		"--no-extensions",
		"--no-skills",
		"--no-prompt-templates",
		"--no-themes",
		"--thinking", "off",
	}
	if len(tools) > 0 {
		args = append(args, "--tools", strings.Join(tools, ","))
	}
	return args
}

func buildSandboxPath(brStub bool, workspace, piDir, goDir, nodeDir string) string {
	parts := []string{}
	if brStub {
		parts = append(parts, filepath.Join(workspace, ".lab-bin"))
	}
	if piDir != "" {
		parts = append(parts, piDir)
	}
	if goDir != "" {
		parts = append(parts, goDir)
	}
	if nodeDir != "" {
		parts = append(parts, nodeDir)
	}
	parts = append(parts, "/usr/local/bin", "/usr/bin", "/bin")
	return strings.Join(parts, ":")
}

// buildDirectEnv builds environment variables for Engine="none".
func buildDirectEnv(workspace, fakeHome, gopath, gomodcache, tmpDir, cacheDir, goRoot, sandboxPath string) []string {
	brLog := filepath.Join(workspace, ".lab-br.log")
	if goRoot == "" {
		goRoot = "/usr/local/go"
	}
	return []string{
		"HOME=" + fakeHome,
		"PATH=" + sandboxPath,
		"USER=sandbox",
		"LANG=C.UTF-8",
		"PI_OFFLINE=1",
		"BR_STUB_LOG=" + brLog,
		"GOPATH=" + gopath,
		"GOMODCACHE=" + gomodcache,
		"GOROOT=" + goRoot,
		"TMPDIR=" + tmpDir,
		"XDG_CACHE_HOME=" + cacheDir,
		"TERM=dumb",
		"SHELL=/bin/bash",
	}
}

func runDiff(scenarioDir, workspace string) (string, error) {
	cmd := exec.Command("diff", "-ruN",
		"--exclude=.lab-bin",
		"--exclude=.lab-br.log",
		scenarioDir, workspace,
	)
	out, err := cmd.Output()
	if err != nil {
		// diff exits 1 when there are differences — that's normal
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return string(out), nil
		}
		return string(out), err
	}
	return string(out), nil
}

// copyDir copies src directory tree into dst (dst will be the root, mirroring src).
func copyDir(src, dst string) error {
	cmd := exec.Command("cp", "-R", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cp -R %s %s: %w\n%s", src, dst, err, out)
	}
	return nil
}

// copyFile copies a single file; silently ignores missing src.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
