package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
)

type MatchRepository struct {
	db *sql.DB
}

func NewMatchRepository(db *sql.DB) *MatchRepository {
	return &MatchRepository{db: db}
}

func (r *MatchRepository) CreateMatch(ctx context.Context, match *model.Match) error {
	_, err := r.db.ExecContext(
		ctx,
		`insert into matches (id, room_id, status, winner_seat, summary, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, $6, $7)`,
		match.ID, match.RoomID, match.Status, match.WinnerSeat, match.Summary, match.CreatedAt, match.UpdatedAt,
	)
	return err
}

func (r *MatchRepository) AppendEvent(ctx context.Context, event *model.MatchEvent) error {
	_, err := r.db.ExecContext(
		ctx,
		`insert into match_events (id, match_id, sequence, event_type, payload, created_at)
		 values ($1, $2, $3, $4, $5, $6)`,
		event.ID, event.MatchID, event.Sequence, event.EventType, event.Payload, event.CreatedAt,
	)
	return err
}

func (r *MatchRepository) CompleteMatch(ctx context.Context, matchID string, winnerSeat *int, status string, summary map[string]any) error {
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(
		ctx,
		`update matches set winner_seat = $2, status = $3, summary = $4, updated_at = now() at time zone 'utc' where id = $1`,
		matchID,
		winnerSeat,
		status,
		summaryJSON,
	)
	return err
}

func (r *MatchRepository) ListHistoryByUser(ctx context.Context, userID string, limit int) ([]model.MatchHistoryItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`select m.id, r.code, m.status, m.created_at, coalesce(m.summary, '{}'::jsonb)
		 from matches m
		 join rooms r on r.id = m.room_id
		 join room_players rp on rp.room_id = r.id
		 where rp.user_id = $1
		 order by m.created_at desc
		 limit $2`,
		userID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.MatchHistoryItem, 0, limit)
	for rows.Next() {
		var item model.MatchHistoryItem
		var summaryRaw []byte
		if err := rows.Scan(&item.MatchID, &item.RoomCode, &item.Result, &item.CreatedAt, &summaryRaw); err != nil {
			return nil, err
		}
		item.Summary = map[string]any{}
		if len(summaryRaw) > 0 {
			if err := json.Unmarshal(summaryRaw, &item.Summary); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
