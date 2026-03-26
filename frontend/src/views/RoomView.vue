<template>
  <main class="shell">
    <ConnectionBanner :show="!roomStore.connected" message="实时连接未建立，正在重连..." />

    <section class="page-header">
      <div>
        <p class="eyebrow">Room</p>
        <h1>房间 {{ code.toUpperCase() }}</h1>
      </div>
      <div class="user-chip">
        <span>{{ currentPlayer?.name ?? auth.user?.username }}</span>
        <button class="ghost-button" @click="leave">离开房间</button>
      </div>
    </section>

    <section v-if="room" class="grid-two">
      <article class="panel">
        <h2>房间状态</h2>
        <div class="stack compact">
          <div class="info-row"><span>状态</span><strong>{{ room.status }}</strong></div>
          <div class="info-row"><span>底分</span><strong>{{ room.settings.baseBet }}</strong></div>
          <div class="info-row"><span>初始积分</span><strong>{{ room.settings.initialScore }}</strong></div>
          <div class="info-row"><span>惩罚</span><strong>{{ room.settings.punishLabel }}</strong></div>
        </div>

        <div class="room-actions">
          <button class="primary-button secondary" @click="toggleReady">
            {{ currentPlayer?.ready ? '取消准备' : '准备' }}
          </button>
          <button class="primary-button" :disabled="!currentPlayer?.host" @click="startGame">房主开始</button>
        </div>
      </article>

      <article class="panel">
        <h2>房间成员</h2>
        <div class="player-list">
          <article v-for="player in room.players" :key="player.id" class="player-row">
            <div>
              <strong>{{ player.name }}</strong>
              <span class="subtle">座位 {{ player.seat + 1 }}</span>
            </div>
            <div class="player-flags">
              <span class="seat-tag" v-if="player.host">房主</span>
              <span class="seat-tag seat-tag-bot" v-if="player.bot">AI</span>
              <span class="seat-tag" :class="player.ready ? 'seat-tag-on' : 'seat-tag-off'">
                {{ player.ready ? '已准备' : '未准备' }}
              </span>
            </div>
          </article>
        </div>
      </article>
    </section>

    <section class="panel">
      <h2>实时消息</h2>
      <div v-if="roomStore.notices.length" class="notice-list">
        <div v-for="notice in roomStore.notices" :key="notice">{{ notice }}</div>
      </div>
      <p v-else class="subtle">等待房主开始游戏。</p>
      <p v-if="roomStore.lastError" class="error-text">{{ roomStore.lastError }}</p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'

import ConnectionBanner from '../components/ConnectionBanner.vue'
import { useAuthStore } from '../stores/auth'
import { useRoomStore } from '../stores/room'

const props = defineProps<{ code: string }>()

const router = useRouter()
const auth = useAuthStore()
const roomStore = useRoomStore()

const room = computed(() => roomStore.room)
const currentPlayer = computed(() =>
  room.value?.players.find((player) => player.userId === auth.user?.id)
)

onMounted(async () => {
  await auth.restore()
  if (roomStore.room?.code !== props.code.toUpperCase()) {
    await roomStore.joinRoom(props.code.toUpperCase())
  }
  if (auth.token) {
    roomStore.connect(auth.token, props.code.toUpperCase())
  }
})

watch(
  () => room.value?.status,
  (status) => {
    if (status === 'playing') {
      router.replace(`/game/${props.code}`)
    }
  },
  { immediate: true }
)

function toggleReady() {
  roomStore.toggleReady(!currentPlayer.value?.ready)
}

function startGame() {
  roomStore.startGame()
}

async function leave() {
  await roomStore.leaveCurrentRoom()
  router.replace('/lobby')
}
</script>
