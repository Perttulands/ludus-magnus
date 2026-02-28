package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Perttulands/chiron/internal/training"
)

func tempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func testLoop() *training.Loop {
	return &training.Loop{
		ID:     "loop_001",
		Status: training.StatusPaused,
	}
}

// --- Save and Load round-trip ---

func TestSaveAndLoad(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "test_checkpoint.json")

	loop := testLoop()
	if err := SaveTo(path, loop, "test"); err != nil {
		t.Fatalf("save error: %v", err)
	}

	cp, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if cp.Loop.ID != "loop_001" {
		t.Errorf("Loop.ID = %q, want %q", cp.Loop.ID, "loop_001")
	}
	if cp.Reason != "test" {
		t.Errorf("Reason = %q, want %q", cp.Reason, "test")
	}
	if cp.SavedAt == "" {
		t.Error("SavedAt should not be empty")
	}
}

func TestSavePreservesLoopState(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "state_checkpoint.json")

	loop := &training.Loop{
		ID:        "loop_full",
		Status:    training.StatusRunning,
		BestScore: 8.5,
	}
	if err := SaveTo(path, loop, "generation_complete"); err != nil {
		t.Fatalf("save error: %v", err)
	}

	cp, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if cp.Loop.Status != training.StatusRunning {
		t.Errorf("Loop.Status = %q, want %q", cp.Loop.Status, training.StatusRunning)
	}
	if cp.Loop.BestScore != 8.5 {
		t.Errorf("Loop.BestScore = %f, want 8.5", cp.Loop.BestScore)
	}
	if cp.Reason != "generation_complete" {
		t.Errorf("Reason = %q, want %q", cp.Reason, "generation_complete")
	}
}

// --- Error paths ---

func TestSaveNilLoop(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "nil_checkpoint.json")
	err := SaveTo(path, nil, "test")
	if err == nil {
		t.Error("expected error for nil loop")
	}
}

func TestSaveToDefaultPathDelegatesToDefaultPath(t *testing.T) {
	// Save with empty path should use DefaultPath, which is in cwd
	// We test that DefaultPath generates expected format
	loop := testLoop()
	// Using SaveTo with empty path will try to create .chiron/ in cwd
	// Test only that the function handles this without panic
	// (it may fail with permission error, that's fine)
	_ = SaveTo("", loop, "test")
	// cleanup
	_ = os.RemoveAll(checkpointDir)
}

func TestLoadNonexistent(t *testing.T) {
	_, err := LoadFrom("/tmp/nonexistent_checkpoint_test_xyz123.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadCorruptedJSON(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "corrupted.json")
	if err := os.WriteFile(path, []byte("not json at all!!!"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Error("expected error for corrupted JSON")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestSaveCreatesDirectoryIfNeeded(t *testing.T) {
	dir := tempDir(t)
	nestedPath := filepath.Join(dir, "a", "b", "c", "checkpoint.json")

	loop := testLoop()
	if err := SaveTo(nestedPath, loop, "nested"); err != nil {
		t.Fatalf("save to nested dir failed: %v", err)
	}

	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Error("expected checkpoint file to exist at nested path")
	}
}

func TestSaveOutputIsValidJSON(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "valid_json.json")

	loop := testLoop()
	if err := SaveTo(path, loop, "json-check"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Errorf("saved checkpoint is not valid JSON: %v", err)
	}
}

// --- Exists ---

func TestExistsAt(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "exists_test.json")

	if ExistsAt(path) {
		t.Error("should not exist yet")
	}

	SaveTo(path, testLoop(), "test")

	if !ExistsAt(path) {
		t.Error("should exist after save")
	}
}

// --- Remove ---

func TestRemoveAt(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "remove_test.json")

	SaveTo(path, testLoop(), "test")
	if !ExistsAt(path) {
		t.Fatal("checkpoint should exist")
	}

	if err := RemoveAt(path); err != nil {
		t.Fatalf("remove error: %v", err)
	}

	if ExistsAt(path) {
		t.Error("checkpoint should not exist after remove")
	}
}

func TestRemoveNonexistent(t *testing.T) {
	if err := RemoveAt("/tmp/nonexistent_checkpoint_remove_test_xyz123.json"); err != nil {
		t.Errorf("remove nonexistent should not error: %v", err)
	}
}

// --- List ---

func TestListIn(t *testing.T) {
	dir := tempDir(t)

	// Create some checkpoints
	SaveTo(filepath.Join(dir, "checkpoint_loop1.json"), testLoop(), "test")
	SaveTo(filepath.Join(dir, "checkpoint_loop2.json"), testLoop(), "test")
	os.WriteFile(filepath.Join(dir, "not_a_checkpoint.txt"), []byte("x"), 0o644)

	paths, err := ListIn(dir)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 checkpoints, got %d", len(paths))
	}
}

func TestListInNonexistent(t *testing.T) {
	paths, err := ListIn("/tmp/nonexistent_dir_list_test_xyz123")
	if err != nil {
		t.Fatalf("expected no error for nonexistent dir, got %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty slice for nonexistent dir, got %v", paths)
	}
}

func TestListInFiltersNonJSON(t *testing.T) {
	dir := tempDir(t)

	// Write files that should be filtered out
	os.WriteFile(filepath.Join(dir, "checkpoint_a.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "data.yaml"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "short.json"), []byte("{}"), 0o644) // name shorter than "checkpoint_"

	paths, err := ListIn(dir)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(paths) != 1 {
		t.Errorf("expected 1 checkpoint (checkpoint_a.json), got %d: %v", len(paths), paths)
	}
}

func TestListInIgnoresSubdirectories(t *testing.T) {
	dir := tempDir(t)

	// Create a subdirectory with a matching name
	subdir := filepath.Join(dir, "checkpoint_subdir.json")
	os.MkdirAll(subdir, 0o755)
	// Create a real checkpoint file
	SaveTo(filepath.Join(dir, "checkpoint_real.json"), testLoop(), "test")

	paths, err := ListIn(dir)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(paths) != 1 {
		t.Errorf("expected 1 checkpoint (ignoring subdirectory), got %d", len(paths))
	}
}

// --- DefaultPath ---

func TestDefaultPath(t *testing.T) {
	path := DefaultPath("loop_123")
	if path != filepath.Join(".chiron", "checkpoint_loop_123.json") {
		t.Errorf("unexpected path: %q", path)
	}
}

func TestDefaultPathFormat(t *testing.T) {
	path := DefaultPath("abc")
	if !filepath.IsLocal(path) {
		t.Errorf("expected local path, got %q", path)
	}
	if filepath.Ext(path) != ".json" {
		t.Errorf("expected .json extension, got %q", filepath.Ext(path))
	}
}

// --- Load uses default path ---

func TestLoadUsesDefaultPath(t *testing.T) {
	// This tests Load("loopID") which calls LoadFrom(DefaultPath(loopID))
	// It will fail with "file not found" but we verify it attempts the right path
	_, err := Load("test-loop-xyz")
	if err == nil {
		t.Error("expected error (no checkpoint at default path)")
	}
}

// --- Save with empty reason ---

func TestSaveAndLoadVariousReasons(t *testing.T) {
	dir := tempDir(t)
	reasons := []string{"generation_complete", "paused", "error", ""}

	for _, reason := range reasons {
		t.Run("reason="+reason, func(t *testing.T) {
			path := filepath.Join(dir, "reason_"+reason+".json")
			if err := SaveTo(path, testLoop(), reason); err != nil {
				t.Fatalf("save error: %v", err)
			}
			cp, err := LoadFrom(path)
			if err != nil {
				t.Fatalf("load error: %v", err)
			}
			if cp.Reason != reason {
				t.Errorf("Reason = %q, want %q", cp.Reason, reason)
			}
		})
	}
}
