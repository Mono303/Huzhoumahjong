package redisrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	client *redis.Client
}

func New(addr, password string, db int) *Repository {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &Repository{client: client}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *Repository) SaveSession(ctx context.Context, user *model.User, ttl time.Duration) error {
	payload, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, sessionKey(user.SessionToken), payload, ttl).Err()
}

func (r *Repository) GetSession(ctx context.Context, sessionToken string) (*model.User, error) {
	value, err := r.client.Get(ctx, sessionKey(sessionToken)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var user model.User
	if err := json.Unmarshal([]byte(value), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) DeleteSession(ctx context.Context, sessionToken string) error {
	return r.client.Del(ctx, sessionKey(sessionToken)).Err()
}

func (r *Repository) SetOnline(ctx context.Context, userID, roomCode string, ttl time.Duration) error {
	return r.client.Set(ctx, onlineKey(userID), roomCode, ttl).Err()
}

func (r *Repository) DeleteOnline(ctx context.Context, userID string) error {
	return r.client.Del(ctx, onlineKey(userID), heartbeatKey(userID), wsKey(userID)).Err()
}

func (r *Repository) RefreshHeartbeat(ctx context.Context, userID, roomCode string, ttl time.Duration) error {
	now := time.Now().UTC().Format(time.RFC3339)
	pipe := r.client.TxPipeline()
	pipe.Set(ctx, heartbeatKey(userID), now, ttl)
	pipe.Set(ctx, onlineKey(userID), roomCode, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *Repository) BindWSConnection(ctx context.Context, userID, roomCode string, ttl time.Duration) error {
	payload := map[string]string{
		"roomCode": roomCode,
		"boundAt":  time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, wsKey(userID), raw, ttl).Err()
}

func (r *Repository) UnbindWSConnection(ctx context.Context, userID string) error {
	return r.client.Del(ctx, wsKey(userID)).Err()
}

func (r *Repository) SaveRoomSnapshot(ctx context.Context, roomCode string, snapshot *model.RoomSnapshot, ttl time.Duration) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	pipe := r.client.TxPipeline()
	pipe.Set(ctx, roomKey(roomCode), payload, ttl)

	playerNames := make([]interface{}, 0, len(snapshot.Players))
	for _, player := range snapshot.Players {
		playerNames = append(playerNames, fmt.Sprintf("%d:%s", player.Seat, player.Name))
	}
	pipe.Del(ctx, roomPlayersKey(roomCode))
	if len(playerNames) > 0 {
		pipe.RPush(ctx, roomPlayersKey(roomCode), playerNames...)
		pipe.Expire(ctx, roomPlayersKey(roomCode), ttl)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func sessionKey(token string) string {
	return "session:" + token
}

func onlineKey(userID string) string {
	return "online:" + userID
}

func heartbeatKey(userID string) string {
	return "heartbeat:" + userID
}

func wsKey(userID string) string {
	return "ws:" + userID
}

func roomKey(roomCode string) string {
	return "room:" + roomCode
}

func roomPlayersKey(roomCode string) string {
	return "room:" + roomCode + ":players"
}
