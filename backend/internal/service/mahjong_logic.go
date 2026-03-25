//go:build ignore
// +build ignore

package service

import (
	"slices"
	"strings"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
)

func isWinningHand(hand []model.Tile, melds []model.Meld) bool {
	if len(hand) == 0 {
		return false
	}
	need := 4 - len(melds)
	if len(melds) == 0 && isSevenPairs(hand) {
		return true
	}
	wildcards := 0
	plainTiles := make([]model.Tile, 0, len(hand))
	for _, tile := range hand {
		if tile.Code == "P" {
			wildcards++
			continue
		}
		plainTiles = append(plainTiles, tile)
	}
	return tryFormMelds(plainTiles, need, wildcards)
}

func isSevenPairs(hand []model.Tile) bool {
	if len(hand) != 14 {
		return false
	}
	counts := tileCounts(hand)
	pairs := 0
	for _, count := range counts {
		if count >= 2 {
			pairs++
		}
	}
	return pairs == 7
}

func tryFormMelds(hand []model.Tile, need int, wildcards int) bool {
	if need == 0 {
		return canMakePair(hand, wildcards)
	}
	if len(hand)+wildcards < need*3+2 {
		return false
	}

	tiles := cloneTiles(hand)
	sortTiles(tiles)
	first := tiles[0]
	sameIndexes := []int{}
	for index, tile := range tiles {
		if tile.Code == first.Code {
			sameIndexes = append(sameIndexes, index)
		}
	}
	if len(sameIndexes) >= 3 && tryFormMelds(removeByIndexes(tiles, sameIndexes[:3]), need-1, wildcards) {
		return true
	}
	if wildcards >= 1 && len(sameIndexes) >= 2 && tryFormMelds(removeByIndexes(tiles, sameIndexes[:2]), need-1, wildcards-1) {
		return true
	}
	if wildcards >= 2 && len(sameIndexes) >= 1 && tryFormMelds(removeByIndexes(tiles, sameIndexes[:1]), need-1, wildcards-2) {
		return true
	}
	if first.Suit != "h" {
		i2 := slices.IndexFunc(tiles[1:], func(tile model.Tile) bool {
			return tile.Suit == first.Suit && tile.Number == first.Number+1
		})
		if i2 >= 0 {
			i2++
		}
		i3 := -1
		if i2 >= 0 {
			offset := slices.IndexFunc(tiles[i2+1:], func(tile model.Tile) bool {
				return tile.Suit == first.Suit && tile.Number == first.Number+2
			})
			if offset >= 0 {
				i3 = i2 + 1 + offset
			}
		}
		if i2 >= 0 && i3 >= 0 && tryFormMelds(removeByIndexes(tiles, []int{0, i2, i3}), need-1, wildcards) {
			return true
		}
		if wildcards >= 1 && i2 >= 0 && tryFormMelds(removeByIndexes(tiles, []int{0, i2}), need-1, wildcards-1) {
			return true
		}
		if wildcards >= 1 && i3 >= 0 && tryFormMelds(removeByIndexes(tiles, []int{0, i3}), need-1, wildcards-1) {
			return true
		}
	}
	return false
}

func canMakePair(hand []model.Tile, wildcards int) bool {
	total := len(hand) + wildcards
	if total != 2 {
		return false
	}
	if wildcards >= 2 {
		return true
	}
	if wildcards == 1 && len(hand) == 1 {
		return true
	}
	return len(hand) == 2 && hand[0].Code == hand[1].Code
}

func removeByIndexes(tiles []model.Tile, indexes []int) []model.Tile {
	remove := map[int]bool{}
	for _, index := range indexes {
		remove[index] = true
	}
	result := make([]model.Tile, 0, len(tiles)-len(indexes))
	for index, tile := range tiles {
		if !remove[index] {
			result = append(result, tile)
		}
	}
	return result
}

func tileCounts(hand []model.Tile) map[string]int {
	counts := map[string]int{}
	for _, tile := range hand {
		counts[tile.Code]++
	}
	return counts
}

func isPureSuit(hand []model.Tile, melds []model.Meld) bool {
	all := append(cloneTiles(hand), flattenMelds(melds)...)
	suits := map[string]bool{}
	for _, tile := range all {
		if tile.Suit == "h" && tile.Code != "P" {
			return false
		}
		if tile.Suit != "h" {
			suits[tile.Suit] = true
		}
	}
	return len(suits) == 1
}

func isMixedSuit(hand []model.Tile, melds []model.Meld) bool {
	all := append(cloneTiles(hand), flattenMelds(melds)...)
	suits := map[string]bool{}
	hasHonor := false
	for _, tile := range all {
		if tile.Suit == "h" && tile.Code != "P" {
			hasHonor = true
			continue
		}
		if tile.Suit != "h" {
			suits[tile.Suit] = true
		}
	}
	return len(suits) == 1 && hasHonor
}

func allTriples(melds []model.Meld) bool {
	if len(melds) == 0 {
		return false
	}
	for _, meld := range melds {
		if meld.Type != "peng" && !strings.HasPrefix(meld.Type, "gang") {
			return false
		}
	}
	return true
}

func flattenMelds(melds []model.Meld) []model.Tile {
	result := []model.Tile{}
	for _, meld := range melds {
		result = append(result, meld.Tiles...)
	}
	return result
}

func (s *RoomService) pickBotDiscardLocked(hand []model.Tile) model.Tile {
	bestIndex := 0
	bestScore := 999
	for index := range hand {
		score := tileValue(hand, index)
		if score < bestScore {
			bestScore = score
			bestIndex = index
		}
	}
	return hand[bestIndex]
}

func tileValue(hand []model.Tile, index int) int {
	tile := hand[index]
	if tile.Code == "P" {
		return 4
	}
	score := 0
	for _, card := range hand {
		if card.Code == tile.Code {
			score += 4
		}
	}
	if tile.Suit != "h" {
		for delta := -2; delta <= 2; delta++ {
			if delta == 0 {
				continue
			}
			target := tile.Number + delta
			for _, card := range hand {
				if card.Suit == tile.Suit && card.Number == target {
					score += 2
					break
				}
			}
		}
	}
	return score
}
