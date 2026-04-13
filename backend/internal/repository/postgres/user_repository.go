package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	"github.com/Mono303/Huzhoumahjong/backend/internal/pkg"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateGuest(ctx context.Context, username, sessionToken string) (*model.User, error) {
	now := time.Now().UTC()
	user := &model.User{
		ID:           pkg.NewID("usr"),
		Username:     username,
		IsGuest:      true,
		SessionToken: sessionToken,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastSeenAt:   now,
	}

	_, err := r.db.ExecContext(
		ctx,
		`insert into users (id, username, is_guest, session_token, created_at, updated_at, last_seen_at)
		 values ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Username, user.IsGuest, user.SessionToken, user.CreatedAt, user.UpdatedAt, user.LastSeenAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetBySessionToken(ctx context.Context, sessionToken string) (*model.User, error) {
	row := r.db.QueryRowContext(
		ctx,
		`select id, username, is_guest, session_token, created_at, updated_at, last_seen_at
		 from users
		 where session_token = $1`,
		sessionToken,
	)
	return scanUser(row)
}

func (r *UserRepository) GetByID(ctx context.Context, userID string) (*model.User, error) {
	row := r.db.QueryRowContext(
		ctx,
		`select id, username, is_guest, session_token, created_at, updated_at, last_seen_at
		 from users
		 where id = $1`,
		userID,
	)
	return scanUser(row)
}

func (r *UserRepository) UpdateLastSeen(ctx context.Context, userID string, at time.Time) error {
	_, err := r.db.ExecContext(
		ctx,
		`update users set last_seen_at = $2, updated_at = $2 where id = $1`,
		userID,
		at.UTC(),
	)
	return err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(row scanner) (*model.User, error) {
	var user model.User
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.IsGuest,
		&user.SessionToken,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastSeenAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
