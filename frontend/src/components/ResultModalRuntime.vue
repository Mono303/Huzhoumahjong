<template>
  <div v-if="result" class="result-mask">
    <div class="result-panel">
      <h3>{{ result.reason }}</h3>
      <p>{{ result.description }}</p>
      <div class="result-grid">
        <div v-for="player in players" :key="player.seat" class="result-item">
          <span>{{ player.name }}</span>
          <strong :class="result.delta[player.seat] >= 0 ? 'plus' : 'minus'">
            {{ result.delta[player.seat] > 0 ? '+' : '' }}{{ result.delta[player.seat] }}
          </strong>
          <small>{{ result.finalScores[player.seat] }} 分</small>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GamePlayer, GameResult } from '../types'

defineProps<{
  result?: GameResult
  players: GamePlayer[]
}>()
</script>
