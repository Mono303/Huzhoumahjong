<template>
  <section class="seat-card" :class="positionClass">
    <header class="seat-head">
      <div>
        <strong>{{ player.name }}</strong>
        <span class="seat-meta">座位 {{ player.seat + 1 }}</span>
      </div>
      <div class="seat-tags">
        <span v-if="player.host" class="seat-tag">房主</span>
        <span v-if="player.bot" class="seat-tag seat-tag-bot">AI</span>
        <span class="seat-tag" :class="player.connected ? 'seat-tag-on' : 'seat-tag-off'">
          {{ player.connected ? '在线' : '离线' }}
        </span>
      </div>
    </header>

    <div class="seat-score">{{ player.score }} 分</div>

    <div class="seat-hand">
      <TileFace
        v-for="tile in visibleTiles"
        :key="tile.key"
        :tile="tile"
        :clickable="clickable && currentTurn"
        :selected="selectedTileKey === tile.key"
        @select="$emit('selectTile', tile.key)"
      />
      <TileFace v-for="index in hiddenCount" :key="`${player.seat}-${index}`" back />
    </div>

    <div v-if="player.melds.length" class="seat-melds">
      <span v-for="meld in player.melds" :key="`${player.seat}-${meld.type}-${meld.tiles[0]?.key}`">
        {{ meldLabel(meld.type) }}
      </span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import TileFace from './TileFace.vue'
import type { GamePlayer } from '../types'

const props = defineProps<{
  player: GamePlayer
  selfSeat: number
  currentTurn: boolean
  clickable?: boolean
  selectedTileKey?: string
  positionClass: string
}>()

defineEmits<{
  (event: 'selectTile', tileKey: string): void
}>()

const visibleTiles = computed(() => props.player.hand ?? [])
const hiddenCount = computed(() => (props.player.hand ? 0 : props.player.handCount))

function meldLabel(type: string) {
  return (
    {
      chi: '吃',
      peng: '碰',
      gang_open: '明杠',
      gang_hidden: '暗杠'
    }[type] ?? type
  )
}
</script>
