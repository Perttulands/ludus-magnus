package state_test

import (
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/state"
)

func TestMigrateStateFromLegacyVersion(t *testing.T) {
	t.Parallel()

	st := state.State{
		Version: "0.9",
	}

	if err := state.MigrateState(&st); err != nil {
		t.Fatalf("migrate state: %v", err)
	}

	if st.Version != state.CurrentVersion {
		t.Fatalf("expected version %q, got %q", state.CurrentVersion, st.Version)
	}

	if st.Sessions == nil {
		t.Fatal("expected sessions map initialized")
	}
}

func TestMigrateStateFailsForUnsupportedVersion(t *testing.T) {
	t.Parallel()

	st := state.State{
		Version: "2.0",
	}

	err := state.MigrateState(&st)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestCompactStateNoop(t *testing.T) {
	t.Parallel()

	before := sampleState()
	after := before

	if err := state.CompactState(&after, 10); err != nil {
		t.Fatalf("compact state: %v", err)
	}

	if after.Version != before.Version {
		t.Fatal("expected compaction to preserve version")
	}
	if len(after.Sessions) != len(before.Sessions) {
		t.Fatal("expected compaction to preserve sessions")
	}
}
