package service

import (
	"context"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
)

type UserRepository interface {
	CreateGuest(ctx context.Context, username, sessionToken string) (*model.User, error)
	GetBySessionToken(ctx context.Context, sessionToken string) (*model.User, error)
	GetByID(ctx context.Context, userID string) (*model.User, error)
	UpdateLastSeen(ctx context.Context, userID string, at time.Time) error
}

type RoomRepository interface {
	CreateRoom(ctx context.Context, room *model.Room, owner *model.RoomPlayer) error
	GetRoomByCode(ctx context.Context, code string) (*model.Room, error)
	ListPlayers(ctx context.Context, roomID string) ([]model.RoomPlayer, error)
	AddPlayer(ctx context.Context, player *model.RoomPlayer) error
	UpdateRoom(ctx context.Context, room *model.Room) error
	UpdateRoomOwner(ctx context.Context, roomID, ownerUserID string) error
	SetPlayerReady(ctx context.Context, roomID, userID string, ready bool) error
	UpdatePlayerConnection(ctx context.Context, roomID, userID string, connected bool) error
	RemovePlayer(ctx context.Context, roomID, userID string) error
}

type MatchRepository interface {
	CreateMatch(ctx context.Context, match *model.Match) error
	AppendEvent(ctx context.Context, event *model.MatchEvent) error
	CompleteMatch(ctx context.Context, matchID string, winnerSeat *int, status string, summary map[string]any) error
	ListHistoryByUser(ctx context.Context, userID string, limit int) ([]model.MatchHistoryItem, error)
}

type CacheRepository interface {
	SaveSession(ctx context.Context, user *model.User, ttl time.Duration) error
	GetSession(ctx context.Context, sessionToken string) (*model.User, error)
	DeleteSession(ctx context.Context, sessionToken string) error
	SetOnline(ctx context.Context, userID, roomCode string, ttl time.Duration) error
	DeleteOnline(ctx context.Context, userID string) error
	RefreshHeartbeat(ctx context.Context, userID, roomCode string, ttl time.Duration) error
	BindWSConnection(ctx context.Context, userID, roomCode string, ttl time.Duration) error
	UnbindWSConnection(ctx context.Context, userID string) error
	SaveRoomSnapshot(ctx context.Context, roomCode string, snapshot *model.RoomSnapshot, ttl time.Duration) error
}

type Notifier interface {
	SendToUser(userID string, envelope model.Envelope)
	BroadcastRoom(roomCode string, envelope model.Envelope)
}
