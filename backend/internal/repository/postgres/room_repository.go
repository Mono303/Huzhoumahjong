package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
)

type RoomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) CreateRoom(ctx context.Context, room *model.Room, owner *model.RoomPlayer) error {
	settingsJSON, err := json.Marshal(room.Settings)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		`insert into rooms (id, code, owner_user_id, status, settings, match_id, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		room.ID, room.Code, room.OwnerUserID, room.Status, settingsJSON, room.MatchID, room.CreatedAt, room.UpdatedAt,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`insert into room_players (id, room_id, user_id, name, seat, is_host, is_ready, is_bot, connected, joined_at)
		 values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		owner.ID, owner.RoomID, owner.UserID, owner.Name, owner.Seat, owner.IsHost, owner.IsReady, owner.IsBot, owner.Connected, owner.JoinedAt,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *RoomRepository) GetRoomByCode(ctx context.Context, code string) (*model.Room, error) {
	row := r.db.QueryRowContext(
		ctx,
		`select id, code, owner_user_id, status, settings, match_id, created_at, updated_at
		 from rooms
		 where code = $1`,
		code,
	)
	return scanRoom(row)
}

func (r *RoomRepository) ListPlayers(ctx context.Context, roomID string) ([]model.RoomPlayer, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`select id, room_id, coalesce(user_id, ''), name, seat, is_host, is_ready, is_bot, connected, joined_at
		 from room_players
		 where room_id = $1
		 order by seat asc`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := make([]model.RoomPlayer, 0, 4)
	for rows.Next() {
		var player model.RoomPlayer
		if err := rows.Scan(
			&player.ID,
			&player.RoomID,
			&player.UserID,
			&player.Name,
			&player.Seat,
			&player.IsHost,
			&player.IsReady,
			&player.IsBot,
			&player.Connected,
			&player.JoinedAt,
		); err != nil {
			return nil, err
		}
		players = append(players, player)
	}
	return players, rows.Err()
}

func (r *RoomRepository) AddPlayer(ctx context.Context, player *model.RoomPlayer) error {
	var userID any
	if player.UserID != "" {
		userID = player.UserID
	}

	_, err := r.db.ExecContext(
		ctx,
		`insert into room_players (id, room_id, user_id, name, seat, is_host, is_ready, is_bot, connected, joined_at)
		 values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		player.ID, player.RoomID, userID, player.Name, player.Seat, player.IsHost, player.IsReady, player.IsBot, player.Connected, player.JoinedAt,
	)
	return err
}

func (r *RoomRepository) UpdateRoom(ctx context.Context, room *model.Room) error {
	settingsJSON, err := json.Marshal(room.Settings)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(
		ctx,
		`update rooms
		 set owner_user_id = $2, status = $3, settings = $4, match_id = $5, updated_at = $6
		 where id = $1`,
		room.ID, room.OwnerUserID, room.Status, settingsJSON, room.MatchID, room.UpdatedAt,
	)
	return err
}

func (r *RoomRepository) UpdateRoomOwner(ctx context.Context, roomID, ownerUserID string) error {
	_, err := r.db.ExecContext(
		ctx,
		`update rooms set owner_user_id = $2, updated_at = now() at time zone 'utc' where id = $1`,
		roomID,
		ownerUserID,
	)
	return err
}

func (r *RoomRepository) SetPlayerReady(ctx context.Context, roomID, userID string, ready bool) error {
	_, err := r.db.ExecContext(
		ctx,
		`update room_players set is_ready = $3 where room_id = $1 and user_id = $2`,
		roomID,
		userID,
		ready,
	)
	return err
}

func (r *RoomRepository) UpdatePlayerConnection(ctx context.Context, roomID, userID string, connected bool) error {
	_, err := r.db.ExecContext(
		ctx,
		`update room_players set connected = $3 where room_id = $1 and user_id = $2`,
		roomID,
		userID,
		connected,
	)
	return err
}

func (r *RoomRepository) RemovePlayer(ctx context.Context, roomID, userID string) error {
	_, err := r.db.ExecContext(
		ctx,
		`delete from room_players where room_id = $1 and user_id = $2`,
		roomID,
		userID,
	)
	return err
}

func scanRoom(row scanner) (*model.Room, error) {
	var room model.Room
	var settingsRaw []byte
	if err := row.Scan(
		&room.ID,
		&room.Code,
		&room.OwnerUserID,
		&room.Status,
		&settingsRaw,
		&room.MatchID,
		&room.CreatedAt,
		&room.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if len(settingsRaw) > 0 {
		if err := json.Unmarshal(settingsRaw, &room.Settings); err != nil {
			return nil, err
		}
	}
	return &room, nil
}
