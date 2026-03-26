import { defineStore } from 'pinia'

import * as roomApi from '../api/rooms'
import type { ActionOption, Envelope, GameSnapshot, MatchHistoryItem, RoomSettings, RoomSnapshot } from '../types'
import { GameSocketRuntime } from '../ws/client_runtime'

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
      return response.room
    },
    async joinRoom(code: string) {
      const response = await roomApi.joinRoom(code)
      this.room = response.room
      return response.room
    },
    async fetchRoom(code: string) {
      const response = await roomApi.getRoom(code)
      this.room = response.room
      return response.room
    },
    async fetchGame(code: string) {
      const response = await roomApi.getGame(code)
      this.game = response.game
      return response.game
    },
    async leaveCurrentRoom() {
      if (!this.room) {
        return
      }
      await roomApi.leaveRoom(this.room.code)
      this.disconnect()
      this.room = null
      this.game = null
    },
    connect(token: string, roomCode: string) {
      if (this.socket && this.room?.code === roomCode) {
        this.socket.connect()
        return
      }
      this.disconnect()
      this.socket = new GameSocketRuntime(roomCode, token, {
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
        case 'room.snapshot':
          this.room = message.payload as RoomSnapshot
          break
        case 'game.snapshot':
          this.game = message.payload as GameSnapshot
          break
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
