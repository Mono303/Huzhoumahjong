import { request } from './http'
import type { MatchHistoryItem, RoomSettings, RoomSnapshot } from '../types'

export function createRoom(settings: RoomSettings) {
  return request<{ room: RoomSnapshot }>('/rooms', {
    method: 'POST',
    body: JSON.stringify({ settings })
  })
}

export function getRoom(code: string) {
  return request<{ room: RoomSnapshot }>(`/rooms/${code}`)
}

export function joinRoom(code: string) {
  return request<{ room: RoomSnapshot }>(`/rooms/${code}/join`, {
    method: 'POST'
  })
}

export function leaveRoom(code: string) {
  return request<{ room: RoomSnapshot }>(`/rooms/${code}/leave`, {
    method: 'POST'
  })
}

export function getHistory() {
  return request<{ items: MatchHistoryItem[] }>('/matches/history')
}
