package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Perttulands/ludus-magnus/pkg/types"
	"github.com/google/uuid"
)

type sessionStore interface {
	CreateSession(ctx context.Context, session types.Session) error
	ListSessions(ctx context.Context) ([]types.Session, error)
}

type Manager struct {
	store sessionStore
}

var allowedModes = map[string]struct{}{
	types.ModeQuickstart: {},
}

func NewManager(store sessionStore) *Manager {
	return &Manager{store: store}
}

func (m *Manager) Create(ctx context.Context, need string, mode string) (types.Session, error) {
	trimmedNeed := strings.TrimSpace(need)
	if trimmedNeed == "" {
		return types.Session{}, fmt.Errorf("need cannot be empty")
	}

	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		normalizedMode = types.ModeQuickstart
	}
	if _, ok := allowedModes[normalizedMode]; !ok {
		return types.Session{}, fmt.Errorf("unsupported mode %q", normalizedMode)
	}

	item := types.Session{
		ID:        uuid.NewString(),
		Need:      trimmedNeed,
		Mode:      normalizedMode,
		CreatedAt: time.Now().UTC(),
	}

	if err := m.store.CreateSession(ctx, item); err != nil {
		return types.Session{}, err
	}

	return item, nil
}

func (m *Manager) List(ctx context.Context) ([]types.Session, error) {
	return m.store.ListSessions(ctx)
}
