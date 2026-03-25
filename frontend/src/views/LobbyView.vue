<template>
  <main class="shell">
    <section class="page-header">
      <div>
        <p class="eyebrow">Lobby</p>
        <h1>大厅</h1>
      </div>
      <div class="user-chip">
        <span>{{ auth.user?.username }}</span>
        <button class="ghost-button" @click="logout">退出</button>
      </div>
    </section>

    <section class="grid-two">
      <article class="panel">
        <h2>创建房间</h2>
        <div class="stack">
          <label class="field">
            <span>初始积分</span>
            <input v-model.number="settings.initialScore" type="number" min="100" step="100" />
          </label>
          <label class="field">
            <span>底分</span>
            <input v-model.number="settings.baseBet" type="number" min="1" step="1" />
          </label>
          <label class="field">
            <span>惩罚文案</span>
            <input v-model.trim="settings.punishLabel" maxlength="12" />
          </label>
          <label class="field">
            <span>惩罚阈值</span>
            <input v-model.number="settings.punishThreshold" type="number" min="0" step="10" />
          </label>
          <button class="primary-button" @click="create">创建房间</button>
        </div>
      </article>

      <article class="panel">
        <h2>加入房间</h2>
        <div class="stack">
          <label class="field">
            <span>房间号</span>
            <input v-model.trim="roomCode" maxlength="8" placeholder="输入房间号" />
          </label>
          <button class="primary-button secondary" @click="join">加入房间</button>
          <p v-if="roomStore.lastError" class="error-text">{{ roomStore.lastError }}</p>
        </div>
      </article>
    </section>

    <section class="panel">
      <div class="panel-title-row">
        <h2>历史战绩</h2>
        <button class="ghost-button" @click="refreshHistory">刷新</button>
      </div>
      <div v-if="roomStore.history.length" class="history-list">
        <article v-for="item in roomStore.history" :key="item.matchId" class="history-item">
          <strong>{{ item.roomCode }}</strong>
          <span>{{ item.result }}</span>
          <small>{{ formatDate(item.createdAt) }}</small>
        </article>
      </div>
      <p v-else class="subtle">还没有历史对局记录。</p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'

import { useAuthStore } from '../stores/auth'
import { useRoomStore } from '../stores/room'

const router = useRouter()
const auth = useAuthStore()
const roomStore = useRoomStore()

const settings = reactive({
  initialScore: 1000,
  baseBet: 10,
  punishLabel: '喝一杯！',
  punishThreshold: 0
})
const roomCode = ref('')

onMounted(async () => {
  await auth.restore()
  await refreshHistory()
})

async function create() {
  try {
    const room = await roomStore.createRoom({ ...settings })
    await router.push(`/room/${room.code}`)
  } catch (err) {
    roomStore.lastError = err instanceof Error ? err.message : '创建房间失败'
  }
}

async function join() {
  if (!roomCode.value) {
    roomStore.lastError = '请输入房间号'
    return
  }
  try {
    const room = await roomStore.joinRoom(roomCode.value.toUpperCase())
    await router.push(`/room/${room.code}`)
  } catch (err) {
    roomStore.lastError = err instanceof Error ? err.message : '加入房间失败'
  }
}

async function refreshHistory() {
  try {
    await roomStore.fetchHistory()
  } catch {
    roomStore.lastError = '获取历史战绩失败'
  }
}

function logout() {
  roomStore.disconnect()
  auth.logout()
  router.replace('/login')
}

function formatDate(raw: string) {
  return new Date(raw).toLocaleString()
}
</script>
