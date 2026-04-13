<template>
  <GamePlayableView :code="code" />
</template>

<script setup lang="ts">
import GamePlayableView from './GamePlayableView.vue'

defineProps<{ code: string }>()
</script>
