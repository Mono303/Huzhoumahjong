package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/config"
	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
	"github.com/Mono303/Huzhoumahjong/backend/internal/pkg"
)

var (
	ErrRoomNotFound        = errors.New("room not found")
	ErrRoomAlreadyPlaying  = errors.New("room is already playing")
	ErrRoomIsFull          = errors.New("room is full")
	ErrPlayerNotInRoom     = errors.New("player is not in room")
	ErrOnlyHostCanStart    = errors.New("only the host can start the game")
	ErrPlayersNotReady     = errors.New("all human players must be ready")
	ErrNotYourTurn         = errors.New("it is not your turn")
	ErrInvalidGameAction   = errors.New("invalid game action")
	ErrGameNotStarted      = errors.New("game has not started")
	ErrRoomRequiresPlayers = errors.New("at least two human players are required")
	ErrLeaveWhilePlaying   = errors.New("cannot leave room while game is in progress")
	ErrReadyWhilePlaying   = errors.New("ready status can only be changed while waiting")
)

type RoomService struct {
	cfg       config.Config
	rooms     map[string]*ActiveRoom
	roomsMu   sync.RWMutex
	roomRepo  RoomRepository
	matchRepo MatchRepository
	cache     CacheRepository
	notifier  Notifier
}

func NewRoomService(
	cfg config.Config,
	roomRepo RoomRepository,
	matchRepo MatchRepository,
	cache CacheRepository,
	notifier Notifier,
) *RoomService {
	return &RoomService{
		cfg:       cfg,
		rooms:     map[string]*ActiveRoom{},
		roomRepo:  roomRepo,
		matchRepo: matchRepo,
		cache:     cache,
		notifier:  notifier,
	}
}

func (s *RoomService) CreateRoom(ctx context.Context, user *model.User, settings model.RoomSettings) (*model.RoomSnapshot, error) {
	if user == nil {
		return nil, errors.New("user is required")
	}
	settings = normalizeSettings(settings)

	room := model.Room{
		ID:          pkg.NewID("room"),
		Code:        pkg.NewRoomCode(s.cfg.RoomCodeLength),
		OwnerUserID: user.ID,
		Status:      model.RoomStatusWaiting,
		Settings:    settings,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	host := model.RoomPlayer{
		ID:        pkg.NewID("rp"),
		RoomID:    room.ID,
		UserID:    user.ID,
		Name:      user.Username,
		Seat:      0,
		IsHost:    true,
		IsReady:   false,
		IsBot:     false,
		Connected: false,
		JoinedAt:  time.Now().UTC(),
	}

	if err := s.roomRepo.CreateRoom(ctx, &room, &host); err != nil {
		return nil, err
	}

	active := &ActiveRoom{
		room:    room,
		players: []model.RoomPlayer{host},
	}

	s.roomsMu.Lock()
	s.rooms[room.Code] = active
	s.roomsMu.Unlock()

	return s.persistAndSnapshot(ctx, active)
}

func (s *RoomService) JoinRoom(ctx context.Context, code string, user *model.User) (*model.RoomSnapshot, error) {
	active, err := s.getOrLoadRoom(ctx, code)
	if err != nil {
		return nil, err
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	if active.room.Status != model.RoomStatusWaiting {
		return nil, ErrRoomAlreadyPlaying
	}
	if s.findPlayerByUserID(active.players, user.ID) != nil {
		return s.snapshotLocked(ctx, active)
	}
	if len(active.players) >= 4 {
		return nil, ErrRoomIsFull
	}

	player := model.RoomPlayer{
		ID:        pkg.NewID("rp"),
		RoomID:    active.room.ID,
		UserID:    user.ID,
		Name:      user.Username,
		Seat:      nextSeat(active.players),
		IsHost:    false,
		IsReady:   false,
		IsBot:     false,
		Connected: false,
		JoinedAt:  time.Now().UTC(),
	}
	if err := s.roomRepo.AddPlayer(ctx, &player); err != nil {
		return nil, err
	}
	active.players = append(active.players, player)
	sortPlayers(active.players)
	active.version++
	s.notifier.BroadcastRoom(active.room.Code, model.Envelope{
		Type: "system.notice",
		Payload: map[string]string{
			"message": fmt.Sprintf("%s joined room %s", user.Username, strings.ToUpper(code)),
		},
	})
	return s.snapshotLocked(ctx, active)
}

func (s *RoomService) LeaveRoom(ctx context.Context, code string, user *model.User) (*model.RoomSnapshot, error) {
	active, err := s.getOrLoadRoom(ctx, code)
	if err != nil {
		return nil, err
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	player := s.findPlayerByUserID(active.players, user.ID)
	if player == nil {
		return nil, ErrPlayerNotInRoom
	}
	if active.room.Status == model.RoomStatusPlaying {
		return nil, ErrLeaveWhilePlaying
	}

	if err := s.roomRepo.RemovePlayer(ctx, active.room.ID, user.ID); err != nil {
		return nil, err
	}

	filtered := make([]model.RoomPlayer, 0, len(active.players)-1)
	for _, candidate := range active.players {
		if candidate.UserID != user.ID {
			filtered = append(filtered, candidate)
		}
	}
	active.players = filtered

	if active.room.OwnerUserID == user.ID {
		nextHost := s.nextHumanPlayer(active.players)
		if nextHost != nil {
			active.room.OwnerUserID = nextHost.UserID
			for index := range active.players {
				active.players[index].IsHost = active.players[index].UserID == nextHost.UserID
			}
			if err := s.roomRepo.UpdateRoomOwner(ctx, active.room.ID, nextHost.UserID); err != nil {
				return nil, err
			}
		}
	}

	active.version++
	active.room.UpdatedAt = time.Now().UTC()
	if len(active.players) == 0 {
		active.room.Status = model.RoomStatusFinished
	}
	if err := s.roomRepo.UpdateRoom(ctx, &active.room); err != nil {
		return nil, err
	}
	return s.snapshotLocked(ctx, active)
}

func (s *RoomService) GetRoomSnapshot(ctx context.Context, code string) (*model.RoomSnapshot, error) {
	active, err := s.getOrLoadRoom(ctx, code)
	if err != nil {
		return nil, err
	}

	active.mu.Lock()
	defer active.mu.Unlock()
	return s.snapshotLocked(ctx, active)
}

func (s *RoomService) ToggleReady(ctx context.Context, code string, user *model.User, ready bool) (*model.RoomSnapshot, error) {
	active, err := s.getOrLoadRoom(ctx, code)
	if err != nil {
		return nil, err
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	player := s.findPlayerByUserID(active.players, user.ID)
	if player == nil {
		return nil, ErrPlayerNotInRoom
	}
	if active.room.Status != model.RoomStatusWaiting {
		return nil, ErrReadyWhilePlaying
	}
	player.IsReady = ready
	for index := range active.players {
		if active.players[index].UserID == user.ID {
			active.players[index].IsReady = ready
		}
	}

	if err := s.roomRepo.SetPlayerReady(ctx, active.room.ID, user.ID, ready); err != nil {
		return nil, err
	}
	active.version++
	active.room.UpdatedAt = time.Now().UTC()
	return s.snapshotLocked(ctx, active)
}

func (s *RoomService) ConnectUser(ctx context.Context, code string, user *model.User) error {
	active, err := s.getOrLoadRoom(ctx, code)
	if err != nil {
		return err
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	player := s.findPlayerByUserID(active.players, user.ID)
	if player == nil {
		return ErrPlayerNotInRoom
	}
	player.Connected = true
	for index := range active.players {
		if active.players[index].UserID == user.ID {
			active.players[index].Connected = true
		}
	}

	_ = s.cache.SetOnline(ctx, user.ID, code, s.cfg.RoomTTL)
	_ = s.cache.BindWSConnection(ctx, user.ID, code, s.cfg.RoomTTL)
	_ = s.cache.RefreshHeartbeat(ctx, user.ID, code, s.cfg.RoomTTL)
	_ = s.roomRepo.UpdatePlayerConnection(ctx, active.room.ID, user.ID, true)
	active.version++
	_, _ = s.snapshotLocked(ctx, active)
	return nil
}

func (s *RoomService) DisconnectUser(ctx context.Context, code string, user *model.User) error {
	active, err := s.getOrLoadRoom(ctx, code)
	if err != nil {
		return err
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	for index := range active.players {
		if active.players[index].UserID == user.ID {
			active.players[index].Connected = false
		}
	}
	_ = s.cache.DeleteOnline(ctx, user.ID)
	_ = s.cache.UnbindWSConnection(ctx, user.ID)
	_ = s.roomRepo.UpdatePlayerConnection(ctx, active.room.ID, user.ID, false)
	active.version++
	_, _ = s.snapshotLocked(ctx, active)
	return nil
}

func (s *RoomService) RefreshHeartbeat(ctx context.Context, code string, user *model.User) {
	_ = s.cache.RefreshHeartbeat(ctx, user.ID, code, s.cfg.RoomTTL)
}

func (s *RoomService) History(ctx context.Context, userID string) ([]model.MatchHistoryItem, error) {
	return s.matchRepo.ListHistoryByUser(ctx, userID, 20)
}

func (s *RoomService) getOrLoadRoom(ctx context.Context, code string) (*ActiveRoom, error) {
	upperCode := strings.ToUpper(strings.TrimSpace(code))
	if upperCode == "" {
		return nil, ErrRoomNotFound
	}

	s.roomsMu.RLock()
	active, ok := s.rooms[upperCode]
	s.roomsMu.RUnlock()
	if ok {
		return active, nil
	}

	room, err := s.roomRepo.GetRoomByCode(ctx, upperCode)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}
	players, err := s.roomRepo.ListPlayers(ctx, room.ID)
	if err != nil {
		return nil, err
	}
	active = &ActiveRoom{
		room:    *room,
		players: players,
	}

	s.roomsMu.Lock()
	s.rooms[upperCode] = active
	s.roomsMu.Unlock()
	return active, nil
}
