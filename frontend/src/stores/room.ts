import { defineStore } from 'pinia'

import * as roomApi from '../api/rooms'
import type { ActionOption, Envelope, GameSnapshot, MatchHistoryItem, RoomSettings, RoomSnapshot } from '../types'
import { GameSocketRuntime } from '../ws/client_runtime'

const GAME_NOT_STARTED_ERROR = 'game has not started'

function normalizeRoomCode(code: string) {
  return code.trim().toUpperCase()
}

function gameBelongsToRoom(game: GameSnapshot | null, roomCode: string) {
  return game?.roomCode?.toUpperCase() === normalizeRoomCode(roomCode)
}

function wait(delayMs: number) {
  return new Promise<void>((resolve) => {
    window.setTimeout(resolve, delayMs)
  })
}

export const useRoomStore = defineStore('room', {
  state: () => ({
    room: null as RoomSnapshot | null,
    game: null as GameSnapshot | null,
    history: [] as MatchHistoryItem[],
    connected: false,
    lastError: '',
    notices: [] as string[],
    socket: null as GameSocketRuntime | null
  }),
  getters: {
    availableActions(state): ActionOption[] {
      return state.game?.availableActions ?? []
    }
  },
  actions: {
    async fetchHistory() {
      const response = await roomApi.getHistory()
      this.history = response.items
    },
    async createRoom(settings: RoomSettings) {
      const response = await roomApi.createRoom(settings)
      this.room = response.room
      this.game = null
      this.lastError = ''
      return response.room
    },
    async joinRoom(code: string) {
      const upperCode = normalizeRoomCode(code)
      const response = await roomApi.joinRoom(upperCode)
      if (!gameBelongsToRoom(this.game, upperCode)) {
        this.game = null
      }
      this.room = response.room
      if (response.room.status !== 'playing') {
        this.game = null
      }
      this.lastError = ''
      return response.room
    },
    async fetchRoom(code: string) {
      const upperCode = normalizeRoomCode(code)
      const response = await roomApi.getRoom(upperCode)
      if (!gameBelongsToRoom(this.game, upperCode)) {
        this.game = null
      }
      this.room = response.room
      if (response.room.status !== 'playing') {
        this.game = null
      }
      this.lastError = ''
      return response.room
    },
    async fetchGame(code: string) {
      const upperCode = normalizeRoomCode(code)
      const response = await roomApi.getGame(upperCode)
      this.game = response.game
      this.lastError = ''
      return response.game
    },
    async ensureGame(code: string, attempts = 8, delayMs = 350) {
      const upperCode = normalizeRoomCode(code)
      if (gameBelongsToRoom(this.game, upperCode)) {
        return this.game
      }

      let lastError: unknown = null
      for (let attempt = 0; attempt < attempts; attempt += 1) {
        try {
          return await this.fetchGame(upperCode)
        } catch (error) {
          lastError = error
          const message = error instanceof Error ? error.message.toLowerCase() : ''
          if (!message.includes(GAME_NOT_STARTED_ERROR) || attempt === attempts - 1) {
            throw error
          }
          await wait(delayMs)
        }
      }

      throw lastError instanceof Error ? lastError : new Error('加载牌局失败')
    },
    async leaveCurrentRoom() {
      if (!this.room) {
        return
      }
      await roomApi.leaveRoom(this.room.code)
      this.disconnect()
      this.room = null
      this.game = null
      this.lastError = ''
      this.notices = []
    },
    connect(token: string, roomCode: string) {
      const upperCode = normalizeRoomCode(roomCode)
      if (this.socket && this.room?.code === upperCode) {
        this.socket.connect()
        return
      }
      this.disconnect()
      this.socket = new GameSocketRuntime(upperCode, token, {
        onOpen: () => {
          this.connected = true
          this.lastError = ''
        },
        onClose: () => {
          this.connected = false
        },
        onError: (message) => {
          this.lastError = message
        },
        onMessage: (message) => {
          this.consume(message)
        }
      })
      this.socket.connect()
    },
    disconnect() {
      this.socket?.disconnect()
      this.socket = null
      this.connected = false
    },
    toggleReady(ready: boolean) {
      this.socket?.send('room.ready', { ready })
    },
    startGame() {
      this.socket?.send('room.start')
    },
    discard(tileKey: string) {
      this.socket?.send('game.discard', { tileKey })
    },
    sendAction(action: ActionOption['type'], tileKey = '', chiIndex = 0) {
      this.socket?.send('game.action', { action, tileKey, chiIndex })
    },
    consume(message: Envelope) {
      switch (message.type) {
        case 'room.snapshot': {
          const snapshot = message.payload as RoomSnapshot
          this.room = snapshot
          if (snapshot.status !== 'playing') {
            this.game = null
          }
          this.lastError = ''
          break
        }
        case 'game.snapshot': {
          const snapshot = message.payload as GameSnapshot
          if (this.room?.code && snapshot.roomCode !== this.room.code) {
            break
          }
          this.game = snapshot
          this.lastError = ''
          break
        }
        case 'system.notice': {
          const payload = message.payload as { message?: string }
          if (payload.message) {
            this.notices = [payload.message, ...this.notices].slice(0, 10)
          }
          break
        }
        case 'error': {
          const payload = message.payload as { message?: string }
          this.lastError = payload.message ?? '操作失败'
          break
        }
        case 'pong':
          break
      }
    }
  }
})
