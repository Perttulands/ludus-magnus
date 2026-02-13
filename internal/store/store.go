package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Perttulands/agent-academy/pkg/types"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    need TEXT NOT NULL,
    mode TEXT NOT NULL,
    created_at TEXT NOT NULL
);
`

type Store struct {
	db *sql.DB
}

func DefaultDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".academy", "academy.db")
	}

	return filepath.Join(configDir, "agent-academy", "academy.db")
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("database path is required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	st := &Store{db: db}
	if err := st.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return st, nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("run schema migration: %w", err)
	}

	return nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Store) CreateSession(ctx context.Context, session types.Session) error {
	const query = `INSERT INTO sessions (id, need, mode, created_at) VALUES (?, ?, ?, ?)`

	_, err := s.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.Need,
		session.Mode,
		session.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

func (s *Store) ListSessions(ctx context.Context) ([]types.Session, error) {
	const query = `SELECT id, need, mode, created_at FROM sessions ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]types.Session, 0)
	for rows.Next() {
		var item types.Session
		var createdAt string
		if err := rows.Scan(&item.ID, &item.Need, &item.Mode, &createdAt); err != nil {
			return nil, fmt.Errorf("scan session row: %w", err)
		}

		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse session created_at: %w", err)
		}
		item.CreatedAt = parsed
		sessions = append(sessions, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session rows: %w", err)
	}

	return sessions, nil
}
