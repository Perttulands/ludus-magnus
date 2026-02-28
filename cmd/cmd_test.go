package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Perttulands/chiron/internal/state"
)

// --- Subcommand routing: verify all subcommands are registered ---

func TestRootCmdHasAllSubcommands(t *testing.T) {
	root := newRootCmd()

	expected := []string{
		"session", "quickstart", "training", "lineage",
		"iterate", "run", "evaluate", "artifact",
		"promote", "directive", "export", "doctor",
	}

	registered := map[string]bool{}
	for _, sub := range root.Commands() {
		registered[sub.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Errorf("expected subcommand %q not registered on root", name)
		}
	}
}

// --- Help output verification ---

func TestRootCmdHelp(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Chiron") {
		t.Errorf("expected help output to contain 'Chiron', got:\n%s", output)
	}
}

func TestSessionCmdHelp(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"session", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("session --help failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Manage sessions") {
		t.Errorf("expected session help, got:\n%s", output)
	}
}

func TestDoctorCmdHelp(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"doctor", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("doctor --help failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "diagnostics") {
		t.Errorf("expected doctor help, got:\n%s", output)
	}
}

func TestExportCmdHelp(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"export", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("export --help failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Export agents") {
		t.Errorf("expected export help, got:\n%s", output)
	}
}

func TestDirectiveCmdHelp(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"directive", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("directive --help failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "directive") {
		t.Errorf("expected directive help, got:\n%s", output)
	}
}

func TestArtifactCmdHelp(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"artifact", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("artifact --help failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "artifact") {
		t.Errorf("expected artifact help, got:\n%s", output)
	}
}

// --- JSON flag ---

func TestGlobalJSONFlag(t *testing.T) {
	root := newRootCmd()
	f := root.PersistentFlags().Lookup("json")
	if f == nil {
		t.Fatal("expected persistent --json flag on root")
	}
	if f.DefValue != "false" {
		t.Errorf("expected --json default false, got %q", f.DefValue)
	}
}

// --- Argument validation ---

func TestRunCmdRequiresArgs(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"run"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	err := root.Execute()
	if err == nil {
		t.Error("expected error for run without session-id arg")
	}
}

func TestEvaluateCmdRequiresArgs(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"evaluate"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	err := root.Execute()
	if err == nil {
		t.Error("expected error for evaluate without artifact-id arg")
	}
}

func TestIterateCmdRequiresArgs(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"iterate"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	err := root.Execute()
	if err == nil {
		t.Error("expected error for iterate without session-id arg")
	}
}

func TestPromoteCmdRequiresArgs(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"promote"})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	err := root.Execute()
	if err == nil {
		t.Error("expected error for promote without session-id arg")
	}
}

// --- Helper function tests ---

func makeTestSession() state.Session {
	return state.Session{
		ID:   "ses_12345678",
		Mode: "quickstart",
		Need: "test",
		Lineages: map[string]state.Lineage{
			"lin_abc": {
				ID:   "lin_abc",
				Name: "main",
				Agents: []state.Agent{
					{ID: "agt_1", Version: 1, Definition: state.AgentDefinition{SystemPrompt: "v1"}},
					{ID: "agt_2", Version: 2, Definition: state.AgentDefinition{SystemPrompt: "v2"}},
				},
			},
		},
	}
}

func TestFindLineageByName(t *testing.T) {
	session := makeTestSession()

	key, lineage, ok := findLineageByName(session, "main")
	if !ok {
		t.Fatal("expected to find lineage 'main'")
	}
	if lineage.Name != "main" {
		t.Errorf("lineage.Name = %q, want 'main'", lineage.Name)
	}
	if key == "" {
		t.Error("expected non-empty lineage key")
	}
}

func TestFindLineageByNameNotFound(t *testing.T) {
	session := makeTestSession()

	_, _, ok := findLineageByName(session, "nonexistent")
	if ok {
		t.Error("expected not to find lineage 'nonexistent'")
	}
}

func TestLatestAgent(t *testing.T) {
	session := makeTestSession()
	_, lineage, _ := findLineageByName(session, "main")

	agent, ok := latestAgent(lineage)
	if !ok {
		t.Fatal("expected to find latest agent")
	}
	if agent.Version != 2 {
		t.Errorf("expected latest agent version 2, got %d", agent.Version)
	}
}

func TestLatestAgentEmptyLineage(t *testing.T) {
	lineage := state.Lineage{Name: "empty", Agents: []state.Agent{}}
	_, ok := latestAgent(lineage)
	if ok {
		t.Error("expected no agent for empty lineage")
	}
}

func TestLatestAgentSingleAgent(t *testing.T) {
	lineage := state.Lineage{
		Agents: []state.Agent{
			{ID: "agt_1", Version: 1},
		},
	}
	agent, ok := latestAgent(lineage)
	if !ok {
		t.Fatal("expected to find agent")
	}
	if agent.Version != 1 {
		t.Errorf("expected version 1, got %d", agent.Version)
	}
}

func TestModelOrDefault(t *testing.T) {
	tests := []struct {
		override string
		fallback string
		want     string
	}{
		{"custom-model", "default-model", "custom-model"},
		{"", "default-model", "default-model"},
		{"  ", "default-model", "default-model"},
		{"", "", ""},
	}
	for _, tt := range tests {
		got := modelOrDefault(tt.override, tt.fallback)
		if got != tt.want {
			t.Errorf("modelOrDefault(%q, %q) = %q, want %q", tt.override, tt.fallback, got, tt.want)
		}
	}
}

func TestNewPrefixedID(t *testing.T) {
	id := newPrefixedID("ses")
	if !strings.HasPrefix(id, "ses_") {
		t.Errorf("expected prefix 'ses_', got %q", id)
	}
	// Should be ses_ + 8 hex chars
	if len(id) != 12 { // "ses_" (4) + 8 hex = 12
		t.Errorf("expected length 12, got %d for %q", len(id), id)
	}

	// IDs should be unique
	id2 := newPrefixedID("ses")
	if id == id2 {
		t.Error("expected unique IDs, got duplicates")
	}
}

func TestNewPrefixedIDVariousPrefixes(t *testing.T) {
	prefixes := []string{"ses", "lin", "agt", "art", "dir"}
	for _, prefix := range prefixes {
		id := newPrefixedID(prefix)
		if !strings.HasPrefix(id, prefix+"_") {
			t.Errorf("expected prefix %q_, got %q", prefix, id)
		}
	}
}

// --- isJSONOutput / writeJSON ---

func TestIsJSONOutputDefault(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{})
	root.ParseFlags([]string{})

	if isJSONOutput(root) {
		t.Error("expected --json to be false by default")
	}
}

func TestWriteJSON(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)

	err := writeJSON(root, map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("writeJSON error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"key": "value"`) {
		t.Errorf("expected JSON output, got:\n%s", output)
	}
}

// --- removeDirectiveByID ---

func TestRemoveDirectiveByID(t *testing.T) {
	directives := []state.Directive{
		{ID: "dir_1", Text: "first"},
		{ID: "dir_2", Text: "second"},
		{ID: "dir_3", Text: "third"},
	}

	// Remove middle
	result, removed := removeDirectiveByID(directives, "dir_2")
	if !removed {
		t.Error("expected directive to be removed")
	}
	if len(result) != 2 {
		t.Errorf("expected 2 remaining, got %d", len(result))
	}

	// Remove nonexistent
	_, removed = removeDirectiveByID(directives, "dir_999")
	if removed {
		t.Error("expected no removal for nonexistent ID")
	}
}

func TestRemoveDirectiveByIDFirst(t *testing.T) {
	directives := []state.Directive{
		{ID: "dir_1", Text: "first"},
		{ID: "dir_2", Text: "second"},
	}
	result, removed := removeDirectiveByID(directives, "dir_1")
	if !removed {
		t.Error("expected removal")
	}
	if len(result) != 1 || result[0].ID != "dir_2" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestRemoveDirectiveByIDLast(t *testing.T) {
	directives := []state.Directive{
		{ID: "dir_1", Text: "first"},
		{ID: "dir_2", Text: "second"},
	}
	result, removed := removeDirectiveByID(directives, "dir_2")
	if !removed {
		t.Error("expected removal")
	}
	if len(result) != 1 || result[0].ID != "dir_1" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestRemoveDirectiveByIDEmpty(t *testing.T) {
	result, removed := removeDirectiveByID(nil, "dir_1")
	if removed {
		t.Error("expected no removal from nil slice")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

// --- variantsForStrategy ---

func TestVariantsForStrategy(t *testing.T) {
	tests := []struct {
		strategy string
		wantN    int
		wantErr  bool
	}{
		{"", 4, false},
		{"variations", 4, false},
		{"alternatives", 4, false},
		{"invalid-thing", 0, true},
	}
	for _, tt := range tests {
		t.Run("strategy="+tt.strategy, func(t *testing.T) {
			variants, err := variantsForStrategy(tt.strategy)
			if (err != nil) != tt.wantErr {
				t.Errorf("variantsForStrategy(%q) error = %v, wantErr %v", tt.strategy, err, tt.wantErr)
			}
			if !tt.wantErr && len(variants) != tt.wantN {
				t.Errorf("expected %d variants, got %d", tt.wantN, len(variants))
			}
		})
	}
}

// --- normalizeDoctorProvider ---

func TestNormalizeDoctorProvider(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "anthropic"},
		{"anthropic", "anthropic"},
		{"ANTHROPIC", "anthropic"},
		{"openai", "openai-compatible"},
		{"openai_compatible", "openai-compatible"},
		{"openrouter", "openai-compatible"},
		{"litellm", "openai-compatible"},
		{"custom", "custom"},
	}
	for _, tt := range tests {
		got := normalizeDoctorProvider(tt.input)
		if got != tt.want {
			t.Errorf("normalizeDoctorProvider(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Session subcommand registration ---

func TestSessionHasSubcommands(t *testing.T) {
	cmd := newSessionCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, expected := range []string{"new", "list", "inspect"} {
		if !subs[expected] {
			t.Errorf("session missing subcommand %q", expected)
		}
	}
}

func TestArtifactHasSubcommands(t *testing.T) {
	cmd := newArtifactCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, expected := range []string{"list", "inspect"} {
		if !subs[expected] {
			t.Errorf("artifact missing subcommand %q", expected)
		}
	}
}

func TestDirectiveHasSubcommands(t *testing.T) {
	cmd := newDirectiveCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, expected := range []string{"set", "clear"} {
		if !subs[expected] {
			t.Errorf("directive missing subcommand %q", expected)
		}
	}
}

func TestExportHasSubcommands(t *testing.T) {
	cmd := newExportCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, expected := range []string{"agent", "evidence"} {
		if !subs[expected] {
			t.Errorf("export missing subcommand %q", expected)
		}
	}
}

func TestTrainingHasSubcommands(t *testing.T) {
	cmd := newTrainingCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, expected := range []string{"init", "iterate"} {
		if !subs[expected] {
			t.Errorf("training missing subcommand %q", expected)
		}
	}
}

func TestLineageHasSubcommands(t *testing.T) {
	cmd := newLineageCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	for _, expected := range []string{"lock", "unlock"} {
		if !subs[expected] {
			t.Errorf("lineage missing subcommand %q", expected)
		}
	}
}

func TestQuickstartHasSubcommands(t *testing.T) {
	cmd := newQuickstartCmd()
	subs := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}
	if !subs["init"] {
		t.Error("quickstart missing subcommand 'init'")
	}
}

// --- Required flag validation ---

func TestRunCmdRequiresInputFlag(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"run", "ses_12345678"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error for missing --input flag")
	}
}

func TestEvaluateCmdRequiresScoreFlag(t *testing.T) {
	root := newRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"evaluate", "art_12345678"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error for missing --score flag")
	}
}

// --- agentVersionForArtifact ---

func TestAgentVersionForArtifact(t *testing.T) {
	lineage := state.Lineage{
		Agents: []state.Agent{
			{ID: "agt_1", Version: 1},
			{ID: "agt_2", Version: 2},
		},
	}

	if v := agentVersionForArtifact(lineage, "agt_1"); v != 1 {
		t.Errorf("expected version 1, got %d", v)
	}
	if v := agentVersionForArtifact(lineage, "agt_2"); v != 2 {
		t.Errorf("expected version 2, got %d", v)
	}
	if v := agentVersionForArtifact(lineage, "agt_999"); v != 0 {
		t.Errorf("expected version 0 for unknown agent, got %d", v)
	}
}
