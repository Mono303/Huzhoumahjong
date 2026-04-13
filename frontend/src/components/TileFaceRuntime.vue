<template>
  <button
    class="tile-face"
    :class="[{ clickable, selected, back }, `tile-${tile?.suit ?? 'back'}`]"
    :disabled="!clickable"
    @click="$emit('select')"
  >
    <span v-if="back">牌背</span>
    <template v-else-if="tile">
      <span class="tile-main">{{ label }}</span>
      <span class="tile-sub">{{ suffix }}</span>
    </template>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type { Tile } from '../types'

const props = defineProps<{
  tile?: Tile
  clickable?: boolean
  selected?: boolean
  back?: boolean
}>()

defineEmits<{
  (event: 'select'): void
}>()

const label = computed(() => {
  if (!props.tile) {
    return ''
  }
  if (props.tile.suit === 'h') {
    return (
      {
        E: '东',
        S: '南',
        W: '西',
        N: '北',
        Z: '中',
        F: '发',
        P: '白'
      }[props.tile.code] ?? props.tile.code
    )
  }
  return String(props.tile.number)
})

const suffix = computed(() => {
  if (!props.tile || props.tile.suit === 'h') {
    return ''
  }
  return (
    {
      w: '万',
      t: '条',
      b: '筒'
    }[props.tile.suit] ?? ''
  )
})
</script>
