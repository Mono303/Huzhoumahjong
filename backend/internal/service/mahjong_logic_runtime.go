package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Mono303/Huzhoumahjong/backend/internal/model"
)

func isWinningHand(hand []model.Tile, melds []model.Meld) bool {
	if len(hand) == 0 {
		return false
	}

	needGroups := 4 - len(melds)
	if len(hand) != needGroups*3+2 {
		return false
	}

	plainTiles, wildcards := splitWildcards(hand)
	if len(melds) == 0 {
		if isSevenPairsWithWildcards(plainTiles, wildcards) {
			return true
		}
		if isThirteenMismatch(plainTiles, wildcards) {
			return true
		}
	}

	counts := tileCounts(plainTiles)
	for _, code := range sortedCountKeys(counts) {
		count := counts[code]
		if count >= 2 {
			counts[code] -= 2
			if counts[code] == 0 {
				delete(counts, code)
			}
			if canCompleteGroups(counts, needGroups, wildcards) {
				return true
			}
			counts[code] = count
		}
		if count >= 1 && wildcards >= 1 {
			counts[code]--
			if counts[code] == 0 {
				delete(counts, code)
			}
			if canCompleteGroups(counts, needGroups, wildcards-1) {
				return true
			}
			counts[code] = count
		}
	}

	return wildcards >= 2 && canCompleteGroups(counts, needGroups, wildcards-2)
}

func isSevenPairs(hand []model.Tile) bool {
	plainTiles, wildcards := splitWildcards(hand)
	return isSevenPairsWithWildcards(plainTiles, wildcards)
}

func isSevenPairsWithWildcards(plainTiles []model.Tile, wildcards int) bool {
	if len(plainTiles)+wildcards != 14 {
		return false
	}

	counts := tileCounts(plainTiles)
	pairs := 0
	singles := 0
	for _, count := range counts {
		pairs += count / 2
		if count%2 == 1 {
			singles++
		}
	}
	if singles > wildcards {
		return false
	}

	wildcards -= singles
	pairs += singles
	pairs += wildcards / 2
	return pairs >= 7
}

func isThirteenMismatch(plainTiles []model.Tile, wildcards int) bool {
	total := len(plainTiles) + wildcards
	if total != 14 {
		return false
	}

	counts := tileCounts(plainTiles)
	for _, count := range counts {
		if count > 1 {
			return false
		}
	}

	perSuit := map[string][]int{}
	honors := map[string]bool{}
	for _, tile := range plainTiles {
		if tile.Suit == "h" {
			if honors[tile.Code] {
				return false
			}
			honors[tile.Code] = true
			continue
		}
		perSuit[tile.Suit] = append(perSuit[tile.Suit], tile.Number)
	}

	for _, numbers := range perSuit {
		sort.Ints(numbers)
		for index := 1; index < len(numbers); index++ {
			if numbers[index]-numbers[index-1] < 3 {
				return false
			}
		}
	}
	return true
}

func isThirteenMismatchWithHand(hand []model.Tile) bool {
	plainTiles, wildcards := splitWildcards(hand)
	return isThirteenMismatch(plainTiles, wildcards)
}

func canCompleteGroups(counts map[string]int, needGroups int, wildcards int) bool {
	if remainingTileCount(counts)+wildcards != needGroups*3 {
		return false
	}
	memo := map[string]bool{}
	return canCompleteGroupsMemo(cloneCounts(counts), needGroups, wildcards, memo)
}

func canCompleteGroupsMemo(counts map[string]int, needGroups int, wildcards int, memo map[string]bool) bool {
	if needGroups == 0 {
		return remainingTileCount(counts) == 0 && wildcards == 0
	}

	key := fmt.Sprintf("%d|%d|%s", needGroups, wildcards, countsKey(counts))
	if result, ok := memo[key]; ok {
		return result
	}

	if remainingTileCount(counts)+wildcards != needGroups*3 {
		memo[key] = false
		return false
	}
	if remainingTileCount(counts) == 0 {
		memo[key] = wildcards == needGroups*3
		return memo[key]
	}

	firstCode := firstCountCode(counts)
	firstTile := tileFromCode(firstCode)
	original := counts[firstCode]

	// Try triplets that consume the first tile.
	for realUsed := minInt(3, original); realUsed >= 1; realUsed-- {
		needWild := 3 - realUsed
		if needWild > wildcards {
			continue
		}
		counts[firstCode] = original - realUsed
		if counts[firstCode] == 0 {
			delete(counts, firstCode)
		}
		if canCompleteGroupsMemo(counts, needGroups-1, wildcards-needWild, memo) {
			counts[firstCode] = original
			memo[key] = true
			return true
		}
		counts[firstCode] = original
	}

	// Try sequences that contain the first tile.
	if firstTile.Suit != "h" {
		for start := firstTile.Number - 2; start <= firstTile.Number; start++ {
			if start < 1 || start+2 > 9 {
				continue
			}

			spentWild := 0
			removed := map[string]int{}
			valid := true
			for number := start; number <= start+2; number++ {
				code := fmt.Sprintf("%s%d", firstTile.Suit, number)
				if counts[code] > removed[code] {
					removed[code]++
					continue
				}
				spentWild++
				if spentWild > wildcards {
					valid = false
					break
				}
			}
			if !valid || removed[firstCode] == 0 {
				continue
			}

			for code, amount := range removed {
				counts[code] -= amount
				if counts[code] == 0 {
					delete(counts, code)
				}
			}
			if canCompleteGroupsMemo(counts, needGroups-1, wildcards-spentWild, memo) {
				for code, amount := range removed {
					counts[code] += amount
				}
				memo[key] = true
				return true
			}
			for code, amount := range removed {
				counts[code] += amount
			}
		}
	}

	memo[key] = false
	return false
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

func countWildcards(hand []model.Tile) int {
	count := 0
	for _, tile := range hand {
		if tile.Code == "P" {
			count++
		}
	}
	return count
}

func countFourOfAKindPairs(hand []model.Tile) int {
	plainTiles, _ := splitWildcards(hand)
	counts := tileCounts(plainTiles)
	total := 0
	for _, count := range counts {
		if count == 4 {
			total++
		}
	}
	return total
}

func splitWildcards(hand []model.Tile) ([]model.Tile, int) {
	plainTiles := make([]model.Tile, 0, len(hand))
	wildcards := 0
	for _, tile := range hand {
		if tile.Code == "P" {
			wildcards++
			continue
		}
		plainTiles = append(plainTiles, tile)
	}
	return plainTiles, wildcards
}

func isBurstHeadLocked(game *GameState, seat int) bool {
	if game == nil || game.LastDrawSeat != seat || game.LastDrawTile == nil || countWildcards(game.Hands[seat]) == 0 {
		return false
	}

	remaining := cloneTiles(game.Hands[seat])
	drawRemoved := false
	filtered := make([]model.Tile, 0, len(remaining)-1)
	for _, tile := range remaining {
		if !drawRemoved && tile.Key == game.LastDrawTile.Key {
			drawRemoved = true
			continue
		}
		filtered = append(filtered, tile)
	}
	if !drawRemoved {
		return false
	}

	wildRemoved := false
	withoutPair := make([]model.Tile, 0, len(filtered)-1)
	for _, tile := range filtered {
		if !wildRemoved && tile.Code == "P" {
			wildRemoved = true
			continue
		}
		withoutPair = append(withoutPair, tile)
	}
	if !wildRemoved {
		return false
	}

	plainTiles, wildcards := splitWildcards(withoutPair)
	return canCompleteGroups(tileCounts(plainTiles), 4-len(game.Melds[seat]), wildcards)
}

func remainingTileCount(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func cloneCounts(source map[string]int) map[string]int {
	cloned := make(map[string]int, len(source))
	for code, count := range source {
		cloned[code] = count
	}
	return cloned
}

func countsKey(counts map[string]int) string {
	keys := sortedCountKeys(counts)
	parts := make([]string, 0, len(keys))
	for _, code := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", code, counts[code]))
	}
	return strings.Join(parts, ",")
}

func sortedCountKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for code := range counts {
		keys = append(keys, code)
	}
	sort.Slice(keys, func(i, j int) bool {
		return tileOrder(tileFromCode(keys[i])) < tileOrder(tileFromCode(keys[j]))
	})
	return keys
}

func firstCountCode(counts map[string]int) string {
	keys := sortedCountKeys(counts)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func tileFromCode(code string) model.Tile {
	switch code {
	case "E", "S", "W", "N", "Z", "F", "P":
		return model.Tile{Code: code, Suit: "h"}
	default:
		return model.Tile{
			Code:   code,
			Suit:   string(code[0]),
			Number: int(code[1] - '0'),
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
