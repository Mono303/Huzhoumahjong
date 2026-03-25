export interface User {
  id: string
  username: string
  isGuest: boolean
  createdAt: string
  updatedAt: string
  lastSeenAt: string
}

export interface RoomSettings {
  initialScore: number
  baseBet: number
  punishLabel: string
  punishThreshold: number
}

export interface RoomPlayer {
  id: string
  userId?: string
  name: string
  seat: number
  host: boolean
  ready: boolean
  bot: boolean
  connected: boolean
}

export interface RoomSnapshot {
  code: string
  status: 'waiting' | 'playing' | 'finished'
  ownerUserId: string
  settings: RoomSettings
  matchId?: string
  players: RoomPlayer[]
  createdAt: string
  updatedAt: string
}

export interface Tile {
  key: string
  code: string
  suit: string
  number: number
}

export interface Meld {
  type: string
  tiles: Tile[]
}

export interface GamePlayer {
  seat: number
  name: string
  score: number
  bot: boolean
  host: boolean
  ready: boolean
  connected: boolean
  hand?: Tile[]
  handCount: number
  melds: Meld[]
}

export interface ActionOption {
  type: 'discard' | 'hu' | 'peng' | 'chi' | 'gang' | 'gang_self' | 'pass'
  tileKeys?: string[]
  chiOptions?: Tile[][]
}

export interface GameLogEntry {
  text: string
  createdAt: string
}

export interface GameResult {
  winnerSeat?: number
  selfDraw: boolean
  fan: number
  description: string
  delta: number[]
  finalScores: number[]
  reason: string
}

export interface DiscardRef {
  seat: number
  tile: Tile
}

export interface GameSnapshot {
  matchId: string
  roomCode: string
  status: string
  phase: string
  selfSeat: number
  dealer: number
  round: number
  currentTurn: number
  deckRemaining: number
  players: GamePlayer[]
  discards: Tile[][]
  lastDiscard?: DiscardRef
  logs: GameLogEntry[]
  availableActions: ActionOption[]
  result?: GameResult
}

export interface MatchHistoryItem {
  matchId: string
  roomCode: string
  result: string
  createdAt: string
  summary: Record<string, unknown>
}

export interface Envelope<T = unknown> {
  type: string
  payload: T
}
