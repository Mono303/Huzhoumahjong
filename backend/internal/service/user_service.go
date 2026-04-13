package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	"github.com/Mono303/Huzhoumahjong/backend/internal/pkg"
)

type UserService struct {
	users      UserRepository
	cache      CacheRepository
	sessionTTL time.Duration
}

func NewUserService(users UserRepository, cache CacheRepository, sessionTTL time.Duration) *UserService {
	return &UserService{
		users:      users,
		cache:      cache,
		sessionTTL: sessionTTL,
	}
}

func (s *UserService) GuestLogin(ctx context.Context, username string) (*model.User, string, error) {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return nil, "", errors.New("username is required")
	}
	if len([]rune(trimmed)) > 12 {
		return nil, "", errors.New("username is too long")
	}

	token := pkg.NewToken()
	user, err := s.users.CreateGuest(ctx, trimmed, token)
	if err != nil {
		return nil, "", err
	}
	if err := s.cache.SaveSession(ctx, user, s.sessionTTL); err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *UserService) Authenticate(ctx context.Context, sessionToken string) (*model.User, error) {
	if sessionToken == "" {
		return nil, nil
	}

	user, err := s.cache.GetSession(ctx, sessionToken)
	if err != nil {
		return nil, err
	}
	if user == nil {
		user, err = s.users.GetBySessionToken(ctx, sessionToken)
		if err != nil || user == nil {
			return user, err
		}
		if err := s.cache.SaveSession(ctx, user, s.sessionTTL); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	user.LastSeenAt = now
	_ = s.users.UpdateLastSeen(ctx, user.ID, now)
	_ = s.cache.SaveSession(ctx, user, s.sessionTTL)
	return user, nil
}

func (s *UserService) GetCurrentUser(ctx context.Context, userID string) (*model.User, error) {
	if userID == "" {
		return nil, nil
	}
	return s.users.GetByID(ctx, userID)
}
