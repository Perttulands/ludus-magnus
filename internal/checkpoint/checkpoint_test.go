package checkpoint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/training"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func testLoop() *training.Loop {
	return &training.Loop{
		ID:     "loop_001",
		Status: training.StatusPaused,
	}
}

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

func TestSaveNilLoop(t *testing.T) {
	dir := tempDir(t)
	path := filepath.Join(dir, "nil_checkpoint.json")
	if err := SaveTo(path, nil, "test"); err == nil {
		t.Error("expected error for nil loop")
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := LoadFrom("/tmp/nonexistent_checkpoint_test.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

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
	if err := RemoveAt("/tmp/nonexistent_checkpoint_remove_test.json"); err != nil {
		t.Errorf("remove nonexistent should not error: %v", err)
	}
}

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
	paths, err := ListIn("/tmp/nonexistent_dir_list_test")
	if err != nil {
		t.Fatalf("expected no error for nonexistent dir, got %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty slice for nonexistent dir, got %v", paths)
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath("loop_123")
	if path != filepath.Join(".ludus-magnus", "checkpoint_loop_123.json") {
		t.Errorf("unexpected path: %q", path)
	}
}
