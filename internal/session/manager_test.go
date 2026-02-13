package session_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Perttulands/agent-academy/internal/session"
	"github.com/Perttulands/agent-academy/internal/store"
	"github.com/Perttulands/agent-academy/pkg/types"
)

func TestManagerCreateAndList(t *testing.T) {
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

	manager := session.NewManager(st)

	created, err := manager.Create(context.Background(), "Run first benchmark", types.ModeQuickstart)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty session ID")
	}

	items, err := manager.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 session, got %d", len(items))
	}
}

func TestManagerRejectsEmptyNeed(t *testing.T) {
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

	manager := session.NewManager(st)

	if _, err := manager.Create(context.Background(), " ", types.ModeQuickstart); err == nil {
		t.Fatal("expected error for empty need")
	}
}
