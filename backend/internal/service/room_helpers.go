package service

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
)

func (s *RoomService) persistAndSnapshot(ctx context.Context, active *ActiveRoom) (*model.RoomSnapshot, error) {
	active.mu.Lock()
	defer active.mu.Unlock()
	return s.snapshotLocked(ctx, active)
}

func (s *RoomService) snapshotLocked(ctx context.Context, active *ActiveRoom) (*model.RoomSnapshot, error) {
	active.room.UpdatedAt = time.Now().UTC()
	if err := s.roomRepo.UpdateRoom(ctx, &active.room); err != nil {
		return nil, err
	}
	snapshot := s.buildRoomSnapshot(active)
	if err := s.cache.SaveRoomSnapshot(ctx, active.room.Code, snapshot, s.cfg.RoomTTL); err != nil {
		return nil, err
	}
	s.notifier.BroadcastRoom(active.room.Code, model.Envelope{
		Type:    "room.snapshot",
		Payload: snapshot,
	})
	if active.game != nil {
		s.broadcastGameLocked(active)
	}
	return snapshot, nil
}

func (s *RoomService) broadcastGameLocked(active *ActiveRoom) {
	for _, player := range active.players {
		if player.IsBot || player.UserID == "" {
			continue
		}
		snapshot := s.buildGameSnapshotLocked(active, player.UserID)
		s.notifier.SendToUser(player.UserID, model.Envelope{
			Type:    "game.snapshot",
			Payload: snapshot,
		})
	}
}

func (s *RoomService) buildRoomSnapshot(active *ActiveRoom) *model.RoomSnapshot {
	players := make([]model.RoomPlayerView, 0, len(active.players))
	for _, player := range active.players {
		players = append(players, model.RoomPlayerView{
			ID:        player.ID,
			UserID:    player.UserID,
			Name:      player.Name,
			Seat:      player.Seat,
			Host:      player.IsHost,
			Ready:     player.IsReady,
			Bot:       player.IsBot,
			Connected: player.Connected,
		})
	}
	return &model.RoomSnapshot{
		Code:        active.room.Code,
		Status:      active.room.Status,
		OwnerUserID: active.room.OwnerUserID,
		Settings:    active.room.Settings,
		MatchID:     active.room.MatchID,
		Players:     players,
		CreatedAt:   active.room.CreatedAt,
		UpdatedAt:   active.room.UpdatedAt,
	}
}

func (s *RoomService) buildGameSnapshotLocked(active *ActiveRoom, userID string) *model.GameSnapshot {
	game := active.game
	if game == nil {
		return nil
	}

	selfSeat := seatForUser(active.players, userID)
	players := make([]model.GamePlayerView, 0, len(active.players))
	for _, player := range active.players {
		entry := model.GamePlayerView{
			Seat:      player.Seat,
			Name:      player.Name,
			Score:     game.Scores[player.Seat],
			Bot:       player.IsBot,
			Host:      player.IsHost,
			Ready:     player.IsReady,
			Connected: player.Connected,
			HandCount: len(game.Hands[player.Seat]),
			Melds:     cloneMelds(game.Melds[player.Seat]),
		}
		if player.Seat == selfSeat || game.Status == "finished" {
			entry.Hand = cloneTiles(game.Hands[player.Seat])
		}
		players = append(players, entry)
	}

	discards := make([][]model.Tile, len(game.Discards))
	for index := range game.Discards {
		discards[index] = cloneTiles(game.Discards[index])
	}

	actions := s.availableActionsLocked(active, selfSeat)
	return &model.GameSnapshot{
		MatchID:          game.Match.ID,
		RoomCode:         active.room.Code,
		Status:           game.Status,
		Phase:            game.Phase,
		SelfSeat:         selfSeat,
		Dealer:           game.Dealer,
		Round:            game.Round,
		CurrentTurn:      game.CurrentTurn,
		DeckRemaining:    len(game.Deck) - game.DrawIndex,
		Players:          players,
		Discards:         discards,
		LastDiscard:      cloneDiscard(game.LastDiscard),
		Logs:             append([]model.GameLogEntry(nil), game.Logs...),
		AvailableActions: actions,
		Result:           cloneResult(game.Result),
	}
}

func (s *RoomService) availableActionsLocked(active *ActiveRoom, seat int) []model.GameActionOption {
	game := active.game
	if game == nil || seat < 0 {
		return nil
	}

	if current := currentPendingClaim(game.Pending); current != nil {
		if current.Seat == seat {
			return cloneActionOptions(current.Options)
		}
		return nil
	}

	if game.Phase == "discard" && game.CurrentTurn == seat {
		options := []model.GameActionOption{
			{Type: model.ActionDiscard, TileKeys: tileKeys(game.Hands[seat])},
		}
		if game.TurnDrawn && s.selfDrawWinLocked(game, seat) {
			options = append([]model.GameActionOption{{Type: model.ActionHu}}, options...)
		}
		if game.TurnDrawn {
			gangs := s.selfGangTilesLocked(game, seat)
			if len(gangs) > 0 {
				options = append(options, model.GameActionOption{
					Type:     model.ActionGangSelf,
					TileKeys: tileKeys(gangs),
				})
			}
		}
		return options
	}
	return nil
}

func normalizeSettings(settings model.RoomSettings) model.RoomSettings {
	if settings.InitialScore <= 0 {
		settings.InitialScore = 1000
	}
	if settings.BaseBet <= 0 {
		settings.BaseBet = 10
	}
	if settings.PunishLabel == "" {
		settings.PunishLabel = "喝一杯！"
	}
	return settings
}

func nextSeat(players []model.RoomPlayer) int {
	used := map[int]bool{}
	for _, player := range players {
		used[player.Seat] = true
	}
	for seat := 0; seat < 4; seat++ {
		if !used[seat] {
			return seat
		}
	}
	return len(players)
}

func sortPlayers(players []model.RoomPlayer) {
	sort.Slice(players, func(i, j int) bool {
		return players[i].Seat < players[j].Seat
	})
}

func seatForUser(players []model.RoomPlayer, userID string) int {
	for _, player := range players {
		if player.UserID == userID {
			return player.Seat
		}
	}
	return -1
}

func (s *RoomService) findPlayerByUserID(players []model.RoomPlayer, userID string) *model.RoomPlayer {
	for index := range players {
		if players[index].UserID == userID {
			return &players[index]
		}
	}
	return nil
}

func (s *RoomService) nextHumanPlayer(players []model.RoomPlayer) *model.RoomPlayer {
	for index := range players {
		if !players[index].IsBot && players[index].UserID != "" {
			return &players[index]
		}
	}
	return nil
}

func playerAtSeat(players []model.RoomPlayer, seat int) *model.RoomPlayer {
	for index := range players {
		if players[index].Seat == seat {
			return &players[index]
		}
	}
	return nil
}

func cloneTiles(tiles []model.Tile) []model.Tile {
	if len(tiles) == 0 {
		return nil
	}
	cloned := make([]model.Tile, len(tiles))
	copy(cloned, tiles)
	return cloned
}

func cloneMelds(melds []model.Meld) []model.Meld {
	if len(melds) == 0 {
		return nil
	}
	cloned := make([]model.Meld, 0, len(melds))
	for _, meld := range melds {
		cloned = append(cloned, model.Meld{
			Type:  meld.Type,
			Tiles: cloneTiles(meld.Tiles),
		})
	}
	return cloned
}

func cloneDiscard(discard *model.DiscardRef) *model.DiscardRef {
	if discard == nil {
		return nil
	}
	copy := *discard
	return &copy
}

func cloneResult(result *model.GameResult) *model.GameResult {
	if result == nil {
		return nil
	}
	copy := *result
	copy.Delta = append([]int(nil), result.Delta...)
	copy.FinalScores = append([]int(nil), result.FinalScores...)
	return &copy
}

func cloneActionOptions(options []model.GameActionOption) []model.GameActionOption {
	cloned := make([]model.GameActionOption, 0, len(options))
	for _, option := range options {
		item := option
		item.TileKeys = append([]string(nil), option.TileKeys...)
		if len(option.ChiOptions) > 0 {
			item.ChiOptions = make([][]model.Tile, 0, len(option.ChiOptions))
			for _, seq := range option.ChiOptions {
				item.ChiOptions = append(item.ChiOptions, cloneTiles(seq))
			}
		}
		cloned = append(cloned, item)
	}
	return cloned
}

func tileKeys(tiles []model.Tile) []string {
	keys := make([]string, 0, len(tiles))
	for _, tile := range tiles {
		keys = append(keys, tile.Key)
	}
	return keys
}

func mustJSON(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}
