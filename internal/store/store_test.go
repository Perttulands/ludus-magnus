package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Perttulands/agent-academy/internal/store"
	"github.com/Perttulands/agent-academy/pkg/types"
)

func TestStoreCreateAndListSessions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "academy.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	session := types.Session{
		ID:        "session-1",
		Need:      "Test persistence",
		Mode:      types.ModeQuickstart,
		CreatedAt: time.Now().UTC().Round(time.Second),
	}

	if err := st.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	items, err := st.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 session, got %d", len(items))
	}

	got := items[0]
	if got.ID != session.ID {
		t.Fatalf("session ID mismatch: want %q, got %q", session.ID, got.ID)
	}
	if got.Need != session.Need {
		t.Fatalf("session need mismatch: want %q, got %q", session.Need, got.Need)
	}
	if got.Mode != session.Mode {
		t.Fatalf("session mode mismatch: want %q, got %q", session.Mode, got.Mode)
	}
}
