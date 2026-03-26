<template>
  <div v-if="show" class="connection-banner">
    {{ message }}
  </div>
</template>

<script setup lang="ts">
defineProps<{
  show: boolean
  message: string
}>()
</script>
