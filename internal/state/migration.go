package state

import "fmt"

const (
	legacyVersionWithoutSchema = "0.9"
	// CurrentVersion is the state schema version used by this binary.
	CurrentVersion = "1.0"
)

// migrationFunc updates state in-place from one version to the next.
type migrationFunc func(*State) error

var migrations = map[string]migrationFunc{
	legacyVersionWithoutSchema: migrateV09ToV10,
}

// MigrateState upgrades state to CurrentVersion in-place.
func MigrateState(st *State) error {
	if st == nil {
		return fmt.Errorf("state is nil")
	}

	version := st.Version
	if version == "" {
		version = legacyVersionWithoutSchema
	}

	for version != CurrentVersion {
		migrate, ok := migrations[version]
		if !ok {
			return fmt.Errorf("unsupported state version %q", version)
		}
		if err := migrate(st); err != nil {
			return fmt.Errorf("migrate from %q: %w", version, err)
		}
		version = st.Version
	}

	if st.Sessions == nil {
		st.Sessions = map[string]Session{}
	}

	return nil
}

func migrateV09ToV10(st *State) error {
	if st.Sessions == nil {
		st.Sessions = map[string]Session{}
	}
	st.Version = CurrentVersion
	return nil
}

// CompactState provides a manual compaction hook for future retention logic.
func CompactState(st *State, artifactRetention int) error {
	if st == nil {
		return fmt.Errorf("state is nil")
	}
	if artifactRetention < 0 {
		return fmt.Errorf("artifact retention must be >= 0")
	}

	// v1: no-op placeholder; future versions can prune old artifacts here.
	return nil
}
