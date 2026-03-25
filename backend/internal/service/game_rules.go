package service

import (
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
)

func (s *RoomService) drawTileLocked(game *GameState, seat int) (model.Tile, bool) {
	if game.DrawIndex >= len(game.Deck) {
		return model.Tile{}, false
	}
	tile := game.Deck[game.DrawIndex]
	game.DrawIndex++
	game.Hands[seat] = append(game.Hands[seat], tile)
	sortTiles(game.Hands[seat])
	return tile, true
}

func (s *RoomService) selfDrawWinLocked(game *GameState, seat int) bool {
	return isWinningHand(game.Hands[seat], game.Melds[seat])
}

func (s *RoomService) selfGangTilesLocked(game *GameState, seat int) []model.Tile {
	counts := map[string]int{}
	first := map[string]model.Tile{}
	for _, tile := range game.Hands[seat] {
		counts[tile.Code]++
		if _, ok := first[tile.Code]; !ok {
			first[tile.Code] = tile
		}
	}

	gangs := []model.Tile{}
	for code, count := range counts {
		if count == 4 {
			gangs = append(gangs, first[code])
			continue
		}
		if count == 1 {
			for _, meld := range game.Melds[seat] {
				if meld.Type == "peng" && len(meld.Tiles) > 0 && meld.Tiles[0].Code == code {
					gangs = append(gangs, first[code])
					break
				}
			}
		}
	}
	return gangs
}

func (s *RoomService) firstSelfGangTileLocked(game *GameState, seat int) *model.Tile {
	gangs := s.selfGangTilesLocked(game, seat)
	if len(gangs) == 0 {
		return nil
	}
	return &gangs[0]
}

func (s *RoomService) selectReactionLocked(active *ActiveRoom, fromSeat int, tile model.Tile) (int, []model.GameActionOption) {
	order := orderedSeatsFrom(fromSeat)
	huCandidates := []int{}
	pengGangCandidates := []int{}
	chiCandidate := -1
	chiOptions := []model.GameActionOption{}

	for _, seat := range order {
		if s.canHuOnDiscardLocked(active.game, seat, tile) {
			huCandidates = append(huCandidates, seat)
			continue
		}
		options := []model.GameActionOption{}
		if s.canGangOnDiscardLocked(active.game, seat, tile) {
			options = append(options, model.GameActionOption{Type: model.ActionGang})
		}
		if s.canPengLocked(active.game, seat, tile) {
			options = append(options, model.GameActionOption{Type: model.ActionPeng})
		}
		if len(options) > 0 {
			pengGangCandidates = append(pengGangCandidates, seat)
		}
		if seat == (fromSeat+1)%4 {
			chiSeqs := s.chiOptionsLocked(active.game, seat, tile, fromSeat)
			if len(chiSeqs) > 0 {
				chiCandidate = seat
				chiOptions = []model.GameActionOption{
					{Type: model.ActionChi, ChiOptions: chiSeqs},
				}
			}
		}
	}

	if len(huCandidates) > 0 {
		return huCandidates[0], []model.GameActionOption{
			{Type: model.ActionHu},
			{Type: model.ActionPass},
		}
	}
	if len(pengGangCandidates) > 0 {
		seat := pengGangCandidates[0]
		options := []model.GameActionOption{}
		if s.canGangOnDiscardLocked(active.game, seat, tile) {
			options = append(options, model.GameActionOption{Type: model.ActionGang})
		}
		if s.canPengLocked(active.game, seat, tile) {
			options = append(options, model.GameActionOption{Type: model.ActionPeng})
		}
		options = append(options, model.GameActionOption{Type: model.ActionPass})
		return seat, options
	}
	if chiCandidate >= 0 {
		return chiCandidate, append(chiOptions, model.GameActionOption{Type: model.ActionPass})
	}
	return -1, nil
}

func (s *RoomService) canPengLocked(game *GameState, seat int, tile model.Tile) bool {
	count := 0
	for _, card := range game.Hands[seat] {
		if card.Code == tile.Code {
			count++
		}
	}
	return count >= 2
}

func (s *RoomService) canGangOnDiscardLocked(game *GameState, seat int, tile model.Tile) bool {
	count := 0
	for _, card := range game.Hands[seat] {
		if card.Code == tile.Code {
			count++
		}
	}
	return count >= 3
}

func (s *RoomService) canHuOnDiscardLocked(game *GameState, seat int, tile model.Tile) bool {
	for _, card := range game.Hands[seat] {
		if card.Code == "P" {
			return false
		}
	}
	hand := append(cloneTiles(game.Hands[seat]), tile)
	return isWinningHand(hand, game.Melds[seat])
}

func (s *RoomService) chiOptionsLocked(game *GameState, seat int, tile model.Tile, fromSeat int) [][]model.Tile {
	if (fromSeat+1)%4 != seat || tile.Suit == "h" {
		return nil
	}

	results := [][]model.Tile{}
	seen := map[string]bool{}
	for delta := -2; delta <= 0; delta++ {
		need := []int{tile.Number + delta, tile.Number + delta + 1, tile.Number + delta + 2}
		valid := true
		for _, number := range need {
			if number < 1 || number > 9 {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}

		remaining := cloneTiles(game.Hands[seat])
		sequence := []model.Tile{tile}
		for _, number := range need {
			if number == tile.Number {
				continue
			}
			index := slices.IndexFunc(remaining, func(card model.Tile) bool {
				return card.Suit == tile.Suit && card.Number == number
			})
			if index < 0 {
				valid = false
				break
			}
			sequence = append(sequence, remaining[index])
			remaining = append(remaining[:index], remaining[index+1:]...)
		}
		if !valid {
			continue
		}
		sortTiles(sequence)
		keyParts := []string{}
		for _, seqTile := range sequence {
			keyParts = append(keyParts, fmt.Sprintf("%s%d", seqTile.Suit, seqTile.Number))
		}
		key := strings.Join(keyParts, "-")
		if seen[key] {
			continue
		}
		seen[key] = true
		results = append(results, sequence)
	}
	return results
}

func (s *RoomService) calcFanLocked(game *GameState, seat int, selfDraw bool) (int, string) {
	hand := cloneTiles(game.Hands[seat])
	melds := game.Melds[seat]
	points := 1
	parts := []string{"基础 1"}

	if isPureSuit(hand, melds) {
		points += 6
		parts = append(parts, "清一色 +6")
	} else if isMixedSuit(hand, melds) {
		points += 3
		parts = append(parts, "混一色 +3")
	}
	if len(melds) == 0 && isSevenPairs(hand) {
		honorPairs := 0
		counts := tileCounts(hand)
		for code, count := range counts {
			if strings.HasPrefix(code, "h") && count >= 2 {
				honorPairs++
			}
		}
		if honorPairs >= 2 {
			points += 8
			parts = append(parts, "豪华七对 +8")
		} else {
			points += 4
			parts = append(parts, "七对 +4")
		}
	} else if len(melds) >= 3 && allTriples(melds) {
		points += 4
		parts = append(parts, "对对胡 +4")
	}
	for _, meld := range melds {
		if meld.Type == "gang_hidden" {
			points += 2
			parts = append(parts, "暗杠 +2")
		} else if meld.Type == "gang_open" {
			points++
			parts = append(parts, "明杠 +1")
		}
	}
	if selfDraw {
		points *= 2
		parts = append(parts, "自摸 x2")
	}
	return points, strings.Join(parts, " / ")
}

func shuffledDeck() []model.Tile {
	deck := make([]model.Tile, 0, 136)
	index := 0
	for _, suit := range []string{"w", "t", "b"} {
		for number := 1; number <= 9; number++ {
			for copy := 0; copy < 4; copy++ {
				deck = append(deck, model.Tile{
					Key:    fmt.Sprintf("%s%d-%02d", suit, number, index),
					Code:   fmt.Sprintf("%s%d", suit, number),
					Suit:   suit,
					Number: number,
				})
				index++
			}
		}
	}
	for _, honor := range []string{"E", "S", "W", "N", "Z", "F", "P"} {
		for copy := 0; copy < 4; copy++ {
			deck = append(deck, model.Tile{
				Key:  fmt.Sprintf("%s-%02d", honor, index),
				Code: honor,
				Suit: "h",
			})
			index++
		}
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func sortTiles(tiles []model.Tile) {
	sort.Slice(tiles, func(i, j int) bool {
		return tileOrder(tiles[i]) < tileOrder(tiles[j])
	})
}

func tileOrder(tile model.Tile) int {
	if tile.Code == "P" {
		return 9999
	}
	if tile.Suit == "h" {
		order := map[string]int{"E": 10, "S": 11, "W": 12, "N": 13, "Z": 14, "F": 15, "P": 16}
		return 1000 + order[tile.Code]
	}
	base := map[string]int{"w": 0, "t": 100, "b": 200}
	return base[tile.Suit] + tile.Number
}

func tileLabel(tile model.Tile) string {
	if tile.Suit == "h" {
		names := map[string]string{"E": "东", "S": "南", "W": "西", "N": "北", "Z": "中", "F": "发", "P": "白"}
		return names[tile.Code]
	}
	suits := map[string]string{"w": "万", "t": "条", "b": "筒"}
	return fmt.Sprintf("%d%s", tile.Number, suits[tile.Suit])
}

func botName(seat int) string {
	names := []string{"甲", "乙", "丙", "丁"}
	if seat >= 0 && seat < len(names) {
		return "机器人" + names[seat]
	}
	return "机器人"
}

func seatName(players []model.RoomPlayer, seat int) string {
	for _, player := range players {
		if player.Seat == seat {
			return player.Name
		}
	}
	return fmt.Sprintf("Seat %d", seat)
}

func orderedSeatsFrom(fromSeat int) []int {
	order := make([]int, 0, 3)
	for step := 1; step < 4; step++ {
		order = append(order, (fromSeat+step)%4)
	}
	return order
}

func actionAllowed(options []model.GameActionOption, action model.PlayerActionType) bool {
	return findActionOption(options, action) != nil
}

func findActionOption(options []model.GameActionOption, action model.PlayerActionType) *model.GameActionOption {
	for index := range options {
		if options[index].Type == action {
			return &options[index]
		}
	}
	return nil
}
