package service

import (
	"context"
	"slices"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	"github.com/Mono303/Huzhoumahjong/backend/internal/pkg"
)

func (s *RoomService) resolveReactionsLocked(ctx context.Context, active *ActiveRoom, fromSeat int, tile model.Tile) {
	claims := s.selectReactionLocked(active, fromSeat, tile, "discard")
	if len(claims) == 0 {
		s.advanceTurnLocked(ctx, active, fromSeat)
		return
	}

	active.game.Pending = &PendingReaction{
		FromSeat: fromSeat,
		Tile:     tile,
		Source:   "discard",
		Claims:   claims,
	}
	active.game.Phase = "react"
	active.version++
	s.schedulePendingBotLocked(active)
}

func (s *RoomService) runBotReaction(code string, seat int, version int64) {
	time.Sleep(700 * time.Millisecond)
	ctx := context.Background()
	active, err := s.getOrLoadRoom(ctx, code)
	if err != nil {
		return
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	current := currentPendingClaim(active.game.Pending)
	if active.version != version || active.game == nil || active.game.Pending == nil || current == nil || current.Seat != seat {
		return
	}
	options := current.Options
	if len(options) == 0 {
		return
	}
	action := options[0]
	tileKey := ""
	if len(action.TileKeys) > 0 {
		tileKey = action.TileKeys[0]
	}
	_ = s.applyPendingActionLocked(ctx, active, seat, action.Type, tileKey, 0)
	_, _ = s.snapshotLocked(ctx, active)
}

func (s *RoomService) advanceTurnLocked(ctx context.Context, active *ActiveRoom, fromSeat int) {
	nextSeat := (fromSeat + 1) % 4
	s.beginTurnLocked(ctx, active, nextSeat)
}

func (s *RoomService) applyPendingActionLocked(ctx context.Context, active *ActiveRoom, seat int, action model.PlayerActionType, tileKey string, chiIndex int) error {
	game := active.game
	pending := game.Pending
	if pending == nil {
		return ErrInvalidGameAction
	}

	current := currentPendingClaim(pending)
	if current == nil || current.Seat != seat || !actionAllowed(current.Options, action) {
		return ErrInvalidGameAction
	}

	clearPending := true
	switch action {
	case model.ActionPass:
		s.appendGameLogLocked(ctx, active, "pass", map[string]any{"seat": seat}, seatName(active.players, seat)+" 选择过牌")
		if s.advancePendingClaimLocked(active) {
			clearPending = false
			break
		}
		s.advanceTurnLocked(ctx, active, pending.FromSeat)
	case model.ActionHu:
		if pending.Source == "discard" {
			s.claimDiscardLocked(game, pending.FromSeat, pending.Tile)
			s.finishGameLocked(ctx, active, seat, false, discardWinReasonLocked(game, seat))
			break
		}
		s.finishGameLocked(ctx, active, seat, false, "抢杠胡")
	case model.ActionPeng:
		s.claimDiscardLocked(game, pending.FromSeat, pending.Tile)
		s.doPengLocked(ctx, active, seat, pending.Tile)
	case model.ActionGang:
		s.claimDiscardLocked(game, pending.FromSeat, pending.Tile)
		s.doGangFromDiscardLocked(ctx, active, seat, pending.Tile)
	case model.ActionChi:
		s.claimDiscardLocked(game, pending.FromSeat, pending.Tile)
		option := findActionOption(current.Options, model.ActionChi)
		if option == nil || chiIndex < 0 || chiIndex >= len(option.ChiOptions) {
			return ErrInvalidGameAction
		}
		s.doChiLocked(ctx, active, seat, pending.Tile, option.ChiOptions[chiIndex])
	default:
		return ErrInvalidGameAction
	}

	if clearPending {
		game.Pending = nil
	}
	return nil
}

func (s *RoomService) doPengLocked(ctx context.Context, active *ActiveRoom, seat int, tile model.Tile) {
	removed := 0
	hand := make([]model.Tile, 0, len(active.game.Hands[seat])-2)
	for _, card := range active.game.Hands[seat] {
		if card.Code == tile.Code && removed < 2 {
			removed++
			continue
		}
		hand = append(hand, card)
	}
	active.game.Hands[seat] = hand
	active.game.Melds[seat] = append(active.game.Melds[seat], model.Meld{
		Type:  "peng",
		Tiles: []model.Tile{tile, tile, tile},
	})
	active.game.CurrentTurn = seat
	active.game.TurnDrawn = false
	active.game.GangBlock[seat] = tile.Code
	active.game.Phase = "discard"
	active.game.LastDiscard = nil
	s.appendGameLogLocked(ctx, active, "peng", map[string]any{"seat": seat, "tile": tile}, seatName(active.players, seat)+" 碰 "+tileLabel(tile))
	if player := playerAtSeat(active.players, seat); player != nil && player.IsBot {
		active.version++
		go s.runBotTurn(active.room.Code, seat, active.version)
	}
}

func (s *RoomService) doChiLocked(ctx context.Context, active *ActiveRoom, seat int, tile model.Tile, seq []model.Tile) {
	remaining := cloneTiles(active.game.Hands[seat])
	for _, seqTile := range seq {
		if seqTile.Key == tile.Key {
			continue
		}
		index := slices.IndexFunc(remaining, func(card model.Tile) bool {
			return card.Key == seqTile.Key
		})
		if index >= 0 {
			remaining = append(remaining[:index], remaining[index+1:]...)
		}
	}
	active.game.Hands[seat] = remaining
	active.game.Melds[seat] = append(active.game.Melds[seat], model.Meld{
		Type:  "chi",
		Tiles: cloneTiles(seq),
	})
	active.game.CurrentTurn = seat
	active.game.TurnDrawn = false
	active.game.GangBlock[seat] = ""
	active.game.Phase = "discard"
	active.game.LastDiscard = nil
	s.appendGameLogLocked(ctx, active, "chi", map[string]any{"seat": seat, "tile": tile}, seatName(active.players, seat)+" 吃 "+tileLabel(tile))
	if player := playerAtSeat(active.players, seat); player != nil && player.IsBot {
		active.version++
		go s.runBotTurn(active.room.Code, seat, active.version)
	}
}

func (s *RoomService) doGangFromDiscardLocked(ctx context.Context, active *ActiveRoom, seat int, tile model.Tile) {
	removed := 0
	hand := make([]model.Tile, 0, len(active.game.Hands[seat])-3)
	for _, card := range active.game.Hands[seat] {
		if card.Code == tile.Code && removed < 3 {
			removed++
			continue
		}
		hand = append(hand, card)
	}
	active.game.Hands[seat] = hand
	active.game.Melds[seat] = append(active.game.Melds[seat], model.Meld{
		Type:  "gang_open",
		Tiles: []model.Tile{tile, tile, tile, tile},
	})
	active.game.GangBlock[seat] = ""
	active.game.LastDiscard = nil
	s.appendGameLogLocked(ctx, active, "gang", map[string]any{"seat": seat, "tile": tile}, seatName(active.players, seat)+" 杠 "+tileLabel(tile))
	s.beginTurnWithSourceLocked(ctx, active, seat, "gang")
}

func (s *RoomService) doSelfGangLocked(ctx context.Context, active *ActiveRoom, seat int, tileKey string) error {
	tileIndex := slices.IndexFunc(active.game.Hands[seat], func(tile model.Tile) bool {
		return tile.Key == tileKey
	})
	if tileIndex < 0 {
		return ErrInvalidGameAction
	}
	target := active.game.Hands[seat][tileIndex]

	pengIndex := -1
	for index, meld := range active.game.Melds[seat] {
		if meld.Type == "peng" && len(meld.Tiles) > 0 && meld.Tiles[0].Code == target.Code {
			pengIndex = index
			break
		}
	}

	if pengIndex >= 0 {
		active.game.Hands[seat] = append(active.game.Hands[seat][:tileIndex], active.game.Hands[seat][tileIndex+1:]...)
		active.game.Melds[seat][pengIndex].Type = "gang_open"
		active.game.Melds[seat][pengIndex].Tiles = append(active.game.Melds[seat][pengIndex].Tiles, target)
	} else {
		matches := 0
		remaining := make([]model.Tile, 0, len(active.game.Hands[seat])-4)
		for _, tile := range active.game.Hands[seat] {
			if tile.Code == target.Code && matches < 4 {
				matches++
				continue
			}
			remaining = append(remaining, tile)
		}
		if matches < 4 {
			return ErrInvalidGameAction
		}
		active.game.Hands[seat] = remaining
		active.game.Melds[seat] = append(active.game.Melds[seat], model.Meld{
			Type:  "gang_hidden",
			Tiles: []model.Tile{target, target, target, target},
		})
	}

	active.game.GangBlock[seat] = ""
	s.appendGameLogLocked(ctx, active, "gang_self", map[string]any{"seat": seat, "tile": target}, seatName(active.players, seat)+" 自杠 "+tileLabel(target))
	s.beginTurnWithSourceLocked(ctx, active, seat, "gang")
	return nil
}

func (s *RoomService) finishGameLocked(ctx context.Context, active *ActiveRoom, winnerSeat int, selfDraw bool, reason string) {
	fan, desc := s.calcFanLocked(active.game, winnerSeat, selfDraw)
	points := fan * active.room.Settings.BaseBet
	delta := []int{0, 0, 0, 0}
	if selfDraw {
		perPlayer := points * 2
		for seat := 0; seat < 4; seat++ {
			if seat == winnerSeat {
				continue
			}
			delta[seat] -= perPlayer
			delta[winnerSeat] += perPlayer
		}
	} else if active.game.LastDiscard != nil {
		delta[active.game.LastDiscard.Seat] -= points * 2
		delta[winnerSeat] += points * 2
	}

	for seat := range active.game.Scores {
		active.game.Scores[seat] += delta[seat]
	}

	winnerSeatCopy := winnerSeat
	active.game.Result = &model.GameResult{
		WinnerSeat:  &winnerSeatCopy,
		SelfDraw:    selfDraw,
		Fan:         fan,
		Description: desc,
		Delta:       delta,
		FinalScores: append([]int(nil), active.game.Scores...),
		Reason:      reason,
	}
	active.game.Status = "finished"
	active.game.Phase = "finished"
	active.room.Status = model.RoomStatusFinished
	active.version++

	s.appendGameLogLocked(ctx, active, "game_finish", map[string]any{
		"winnerSeat": winnerSeat,
		"selfDraw":   selfDraw,
		"fan":        fan,
		"delta":      delta,
	}, seatName(active.players, winnerSeat)+" "+reason+"，"+desc)

	_ = s.matchRepo.CompleteMatch(ctx, active.game.Match.ID, &winnerSeatCopy, "finished", map[string]any{
		"winnerSeat": winnerSeat,
		"selfDraw":   selfDraw,
		"fan":        fan,
		"delta":      delta,
		"scores":     active.game.Scores,
		"reason":     reason,
	})
}

func (s *RoomService) finishDrawGameLocked(ctx context.Context, active *ActiveRoom) {
	active.game.Status = "finished"
	active.game.Phase = "finished"
	active.game.Result = &model.GameResult{
		SelfDraw:    false,
		Fan:         0,
		Description: "流局",
		Delta:       []int{0, 0, 0, 0},
		FinalScores: append([]int(nil), active.game.Scores...),
		Reason:      "流局",
	}
	active.room.Status = model.RoomStatusFinished
	active.version++
	s.appendGameLogLocked(ctx, active, "game_draw", map[string]any{}, "本局流局")
	_ = s.matchRepo.CompleteMatch(ctx, active.game.Match.ID, nil, "draw", map[string]any{
		"reason": "draw",
		"scores": active.game.Scores,
	})
}

func (s *RoomService) appendGameLogLocked(ctx context.Context, active *ActiveRoom, eventType string, payload map[string]any, text string) {
	entry := model.GameLogEntry{
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}
	active.game.Logs = append(active.game.Logs, entry)
	if len(active.game.Logs) > 18 {
		active.game.Logs = active.game.Logs[len(active.game.Logs)-18:]
	}
	active.game.EventSeq++
	event := model.MatchEvent{
		ID:        pkg.NewID("evt"),
		MatchID:   active.game.Match.ID,
		Sequence:  active.game.EventSeq,
		EventType: eventType,
		Payload:   mustJSON(payload),
		CreatedAt: entry.CreatedAt,
	}
	_ = s.matchRepo.AppendEvent(ctx, &event)
}

func currentPendingClaim(pending *PendingReaction) *PendingClaim {
	if pending == nil || pending.Index < 0 || pending.Index >= len(pending.Claims) {
		return nil
	}
	return &pending.Claims[pending.Index]
}

func (s *RoomService) advancePendingClaimLocked(active *ActiveRoom) bool {
	if active.game == nil || active.game.Pending == nil {
		return false
	}
	active.game.Pending.Index++
	if currentPendingClaim(active.game.Pending) == nil {
		active.game.Pending = nil
		return false
	}
	active.version++
	s.schedulePendingBotLocked(active)
	return true
}

func (s *RoomService) schedulePendingBotLocked(active *ActiveRoom) {
	current := currentPendingClaim(active.game.Pending)
	if current == nil {
		return
	}
	player := playerAtSeat(active.players, current.Seat)
	if player != nil && player.IsBot {
		version := active.version
		go s.runBotReaction(active.room.Code, current.Seat, version)
	}
}

func (s *RoomService) claimDiscardLocked(game *GameState, fromSeat int, tile model.Tile) {
	if fromSeat < 0 || fromSeat >= len(game.Discards) || len(game.Discards[fromSeat]) == 0 {
		game.LastDiscard = nil
		return
	}

	last := len(game.Discards[fromSeat]) - 1
	if game.Discards[fromSeat][last].Key == tile.Key {
		game.Discards[fromSeat] = game.Discards[fromSeat][:last]
	} else {
		for index := last; index >= 0; index-- {
			if game.Discards[fromSeat][index].Key == tile.Key {
				game.Discards[fromSeat] = append(game.Discards[fromSeat][:index], game.Discards[fromSeat][index+1:]...)
				break
			}
		}
	}
}

func discardWinReasonLocked(game *GameState, seat int) string {
	if isEarthHuLocked(game, seat, "discard") {
		return "地胡"
	}
	return "点炮"
}
