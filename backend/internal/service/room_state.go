package service

import (
	"sync"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
)

type ActiveRoom struct {
	mu      sync.Mutex
	room    model.Room
	players []model.RoomPlayer
	game    *GameState
	version int64
}

type GameState struct {
	Match        model.Match
	RoomCode     string
	Status       string
	Phase        string
	Dealer       int
	Round        int
	CurrentTurn  int
	TurnDrawn    bool
	Deck         []model.Tile
	DrawIndex    int
	Hands        [][]model.Tile
	Melds        [][]model.Meld
	Discards     [][]model.Tile
	Scores       []int
	DiscardCount int
	LastDiscard  *model.DiscardRef
	LastDrawTile *model.Tile
	LastDrawSeat int
	LastDrawVia  string
	GangBlock    []string
	Pending      *PendingReaction
	Logs         []model.GameLogEntry
	Result       *model.GameResult
	EventSeq     int
	LastMoveTime time.Time
}

type PendingReaction struct {
	FromSeat int
	Tile     model.Tile
	Source   string
	Index    int
	Claims   []PendingClaim
}

type PendingClaim struct {
	Seat    int
	Options []model.GameActionOption
}
