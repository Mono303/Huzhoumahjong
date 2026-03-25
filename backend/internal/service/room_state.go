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
	Deck         []model.Tile
	DrawIndex    int
	Hands        [][]model.Tile
	Melds        [][]model.Meld
	Discards     [][]model.Tile
	Scores       []int
	LastDiscard  *model.DiscardRef
	Pending      *PendingReaction
	Logs         []model.GameLogEntry
	Result       *model.GameResult
	EventSeq     int
	LastMoveTime time.Time
}

type PendingReaction struct {
	FromSeat int
	Seat     int
	Tile     model.Tile
	Options  []model.GameActionOption
}
