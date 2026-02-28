package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	stateDirName       = ".chiron"
	legacyStateDirName = ".ludus-magnus"
	stateFileName      = "state.json"
)

// DefaultStatePath returns the default on-disk state location.
func DefaultStatePath() string {
	return filepath.Join(stateDirName, stateFileName)
}

// MigrateLegacyDir renames .ludus-magnus/ to .chiron/ if the old directory
// exists and the new one does not. Returns true if migration occurred.
func MigrateLegacyDir() (bool, error) {
	_, errNew := os.Stat(stateDirName)
	_, errOld := os.Stat(legacyStateDirName)

	if os.IsNotExist(errNew) && errOld == nil {
		if err := os.Rename(legacyStateDirName, stateDirName); err != nil {
			return false, fmt.Errorf("migrate %s to %s: %w", legacyStateDirName, stateDirName, err)
		}
		return true, nil
	}
	return false, nil
}

// Load reads and decodes state from disk.
func Load(path string) (State, error) {
	if path == "" {
		migrated, err := MigrateLegacyDir()
		if err != nil {
			return State{}, fmt.Errorf("legacy state migration: %w", err)
		}
		if migrated {
			fmt.Fprintf(os.Stderr, "Migrated state directory: .ludus-magnus/ -> .chiron/\n")
		}
		path = DefaultStatePath()
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewState(), nil
		}
		return State{}, fmt.Errorf("read state file %q: %w", path, err)
	}

	var st State
	if err := json.Unmarshal(content, &st); err != nil {
		return State{}, fmt.Errorf("decode state file %q: %w", path, err)
	}

	if err := MigrateState(&st); err != nil {
		return State{}, fmt.Errorf("migrate state file %q: %w", path, err)
	}

	return st, nil
}

// Save encodes and writes state to disk.
func Save(path string, st State) error {
	if path == "" {
		path = DefaultStatePath()
	}

	if st.Version == "" {
		st.Version = CurrentVersion
	}
	if st.Sessions == nil {
		st.Sessions = map[string]Session{}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state directory for %q: %w", path, err)
	}

	content, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	content = append(content, '\n')

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write state file %q: %w", path, err)
	}

	return nil
}
