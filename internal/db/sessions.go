package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Session is a login. The browser keeps its token in a cookie.
type Session = SessionByTokenRow

// SessionByToken returns the session with token, or nil if there's none.
func (db *DB) SessionByToken(ctx context.Context, token string) (*Session, error) {
	session, err := db.queries.SessionByToken(ctx, token)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find session: %w", err)
	}
	return &session, nil
}

func (db *DB) CreateSession(ctx context.Context, token string) error {
	if err := db.queries.CreateSession(ctx, token); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// TouchSession sets a session's last activity to now.
func (db *DB) TouchSession(ctx context.Context, id int) error {
	if err := db.queries.TouchSession(ctx, id); err != nil {
		return fmt.Errorf("update session activity: %w", err)
	}
	return nil
}

func (db *DB) DeleteSession(ctx context.Context, token string) error {
	if err := db.queries.DeleteSession(ctx, token); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
