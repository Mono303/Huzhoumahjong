<template>
  <main class="game-shell">
    <ConnectionBanner :show="!roomStore.connected" message="实时连接断开，正在尝试重连..." />
    <ResultModalRuntime :result="game?.result" :players="players" />

    <header class="game-topbar">
      <div>
        <p class="eyebrow">Game</p>
        <h1>房间 {{ code.toUpperCase() }}</h1>
      </div>
      <div class="top-metrics" v-if="game">
        <span>庄家 {{ dealerName }}</span>
        <span>剩余 {{ game.deckRemaining }} 张</span>
        <span>当前回合 Seat {{ game.currentTurn + 1 }}</span>
      </div>
    </header>

    <section v-if="loading" class="panel">
      <h2>正在加载牌局</h2>
      <p class="subtle">房间已经进入对局，正在同步牌桌状态...</p>
    </section>

    <section v-else-if="!game" class="panel stack">
      <div>
        <h2>牌局暂时还没准备好</h2>
        <p class="subtle">如果房主刚刚点击开始游戏，通常几秒内就会同步完成。</p>
      </div>
      <div class="room-actions">
        <button class="primary-button" @click="reloadGame">重新加载</button>
        <button class="primary-button secondary" @click="router.replace(`/room/${props.code}`)">返回房间</button>
      </div>
      <p v-if="roomStore.lastError" class="error-text">{{ roomStore.lastError }}</p>
    </section>

    <template v-else>
      <section class="table-layout">
        <PlayerSeatRuntime
          v-for="entry in positionedPlayers"
          :key="entry.player.seat"
          :player="entry.player"
          :self-seat="game.selfSeat"
          :current-turn="game.currentTurn === entry.player.seat"
          :clickable="entry.player.seat === game.selfSeat && canDiscard"
          :selected-tile-key="selectedTileKey"
          :position-class="entry.position"
          @select-tile="selectTile"
        />

        <section class="table-center">
          <div class="discard-groups">
            <div v-for="entry in positionedPlayers" :key="`${entry.player.seat}-discards`" class="discard-group">
              <strong>{{ entry.player.name }}</strong>
              <div class="discard-row">
                <TileFaceRuntime v-for="tile in game.discards[entry.player.seat]" :key="tile.key" :tile="tile" />
              </div>
            </div>
          </div>

          <div class="action-bar">
            <button
              v-for="action in visibleActions"
              :key="action.type"
              class="primary-button secondary"
              @click="runAction(action.type)"
            >
              {{ actionLabel(action.type) }}
            </button>
            <div v-if="chiAction?.chiOptions?.length" class="chi-picker">
              <button
                v-for="(option, index) in chiAction.chiOptions"
                :key="option.map((tile) => tile.key).join('-')"
                class="ghost-button"
                @click="roomStore.sendAction('chi', '', index)"
              >
                {{ option.map(tileText).join(' ') }}
              </button>
            </div>
          </div>
        </section>
      </section>

      <section class="panel game-panels">
        <div>
          <h2>操作提示</h2>
          <p class="subtle">{{ helperText }}</p>
          <p v-if="roomStore.lastError" class="error-text">{{ roomStore.lastError }}</p>
        </div>
        <div>
          <h2>对局日志</h2>
          <div class="notice-list log-list">
            <div v-for="item in game.logs.slice().reverse()" :key="`${item.createdAt}-${item.text}`">
              {{ item.text }}
            </div>
          </div>
        </div>
      </section>
    </template>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import ConnectionBanner from '../components/ConnectionBanner.vue'
import PlayerSeatRuntime from '../components/PlayerSeatRuntime.vue'
import ResultModalRuntime from '../components/ResultModalRuntime.vue'
import TileFaceRuntime from '../components/TileFaceRuntime.vue'
import { useAuthStore } from '../stores/auth'
import { useRoomStore } from '../stores/room'
import type { ActionOption } from '../types'

const props = defineProps<{ code: string }>()

const router = useRouter()
const auth = useAuthStore()
const roomStore = useRoomStore()

const loading = ref(true)
const selectedTileKey = ref('')

const game = computed(() => roomStore.game)
const room = computed(() => roomStore.room)
const players = computed(() => game.value?.players ?? [])
const canDiscard = computed(() => roomStore.availableActions.some((action) => action.type === 'discard'))
const chiAction = computed(() => roomStore.availableActions.find((action) => action.type === 'chi'))

const positionedPlayers = computed(() => {
  if (!game.value) {
    return []
  }
  const selfSeat = game.value.selfSeat
  const positions = ['bottom', 'right', 'top', 'left']
  return players.value.map((player) => ({
    player,
    position: positions[(player.seat - selfSeat + 4) % 4]
  }))
})

const visibleActions = computed(() =>
  roomStore.availableActions.filter((action) => !['discard', 'chi'].includes(action.type))
)

const dealerName = computed(() => players.value.find((player) => player.seat === game.value?.dealer)?.name ?? '-')

const helperText = computed(() => {
  if (!game.value) {
    return '正在等待牌局快照同步。'
  }
  if (game.value.result) {
    return '本局已经结束，可以返回房间等待下一局。'
  }
  if (canDiscard.value) {
    return selectedTileKey.value
      ? '再次点击已选中的牌即可出牌，也可以直接点操作按钮。'
      : '轮到你了，点击手牌出牌。'
  }
  if (roomStore.availableActions.length) {
    return '你有可响应的动作，请在操作栏中选择。'
  }
  return '等待其他玩家行动。'
})

onMounted(async () => {
  await auth.restore()
  await loadGame()
})

watch(
  () => room.value?.status,
  async (status) => {
    if (status === 'playing' && !game.value) {
      await reloadGame()
    }
    if (status === 'waiting') {
      router.replace(`/room/${props.code}`)
    }
  },
  { immediate: true }
)

watch(
  () => game.value?.availableActions,
  () => {
    selectedTileKey.value = ''
  }
)

async function loadGame() {
  loading.value = true
  try {
    const upperCode = props.code.toUpperCase()
    const currentRoom = await roomStore.fetchRoom(upperCode)
    if (auth.token) {
      roomStore.connect(auth.token, upperCode)
    }
    if (currentRoom.status === 'playing') {
      await roomStore.fetchGame(upperCode)
    }
  } catch (error) {
    roomStore.lastError = error instanceof Error ? error.message : '加载牌局失败'
  } finally {
    loading.value = false
  }
}

async function reloadGame() {
  await loadGame()
}

function selectTile(tileKey: string) {
  if (!canDiscard.value) {
    return
  }
  if (selectedTileKey.value === tileKey) {
    roomStore.discard(tileKey)
    selectedTileKey.value = ''
    return
  }
  selectedTileKey.value = tileKey
}

function runAction(action: string) {
  if (action === 'gang_self') {
    const option = roomStore.availableActions.find((item) => item.type === 'gang_self')
    roomStore.sendAction(action as ActionOption['type'], option?.tileKeys?.[0] ?? '')
    return
  }
  roomStore.sendAction(action as ActionOption['type'])
}

function actionLabel(action: string) {
  return (
    {
      hu: '胡',
      peng: '碰',
      gang: '杠',
      gang_self: '自杠',
      pass: '过'
    }[action] ?? action
  )
}

function tileText(tile: { suit: string; number: number; code: string }) {
  if (tile.suit === 'h') {
    return (
      {
        E: '东',
        S: '南',
        W: '西',
        N: '北',
        Z: '中',
        F: '发',
        P: '白'
      }[tile.code] ?? tile.code
    )
  }
  return `${tile.number}${{ w: '万', t: '条', b: '筒' }[tile.suit as 'w' | 't' | 'b']}`
}
</script>
