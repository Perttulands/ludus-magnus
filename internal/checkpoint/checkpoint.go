package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/training"
)

const (
	checkpointDir  = ".ludus-magnus"
	checkpointFile = "checkpoint.json"
)

// Checkpoint captures the full state of a training loop for resume.
type Checkpoint struct {
	Loop      training.Loop `json:"loop"`
	SavedAt   string        `json:"saved_at"`
	Reason    string        `json:"reason"` // "generation_complete", "paused", "error"
}

// Save persists a training loop checkpoint to disk.
func Save(loop *training.Loop, reason string) error {
	return SaveTo("", loop, reason)
}

// SaveTo persists a checkpoint to a specific path.
func SaveTo(path string, loop *training.Loop, reason string) error {
	if loop == nil {
		return fmt.Errorf("loop is nil")
	}

	if path == "" {
		path = DefaultPath(loop.ID)
	}

	cp := Checkpoint{
		Loop:    *loop,
		SavedAt: time.Now().UTC().Format(time.RFC3339),
		Reason:  reason,
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create checkpoint directory: %w", err)
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("encode checkpoint: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}

	return nil
}

// Load reads a checkpoint from the default location.
func Load(loopID string) (*Checkpoint, error) {
	return LoadFrom(DefaultPath(loopID))
}

// LoadFrom reads a checkpoint from a specific path.
func LoadFrom(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint %q: %w", path, err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("decode checkpoint %q: %w", path, err)
	}

	return &cp, nil
}

// Exists checks whether a checkpoint file exists for a given loop.
func Exists(loopID string) bool {
	return ExistsAt(DefaultPath(loopID))
}

// ExistsAt checks whether a checkpoint file exists at a path.
func ExistsAt(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Remove deletes a checkpoint file.
func Remove(loopID string) error {
	return RemoveAt(DefaultPath(loopID))
}

// RemoveAt deletes a checkpoint at a specific path.
func RemoveAt(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove checkpoint %q: %w", path, err)
	}
	return nil
}

// DefaultPath returns the standard checkpoint location for a loop.
func DefaultPath(loopID string) string {
	return filepath.Join(checkpointDir, fmt.Sprintf("checkpoint_%s.json", loopID))
}

// List returns all checkpoint files found in the default directory.
func List() ([]string, error) {
	return ListIn(checkpointDir)
}

// ListIn returns all checkpoint files in a directory.
// Returns empty slice (not error) when directory does not exist.
func ListIn(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read checkpoint dir %q: %w", dir, err)
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" && len(entry.Name()) > len("checkpoint_") {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	return paths, nil
}
