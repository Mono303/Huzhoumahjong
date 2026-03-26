import type { Envelope } from '../types'

interface SocketHandlers {
  onOpen?: () => void
  onClose?: () => void
  onError?: (message: string) => void
  onMessage?: (message: Envelope) => void
}

export class GameSocketRuntime {
  private socket: WebSocket | null = null
  private reconnectTimer: number | null = null
  private heartbeatTimer: number | null = null
  private intentionalClose = false

  constructor(
    private readonly roomCode: string,
    private readonly token: string,
    private readonly handlers: SocketHandlers
  ) {}

  connect() {
    if (this.socket && this.socket.readyState <= WebSocket.OPEN) {
      return
    }

    const configured = import.meta.env.VITE_WS_BASE_URL
    const base =
      configured ||
      `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/ws`
    const url = new URL(base)
    url.searchParams.set('roomCode', this.roomCode)
    url.searchParams.set('token', this.token)

    this.intentionalClose = false
    this.socket = new WebSocket(url.toString())
    this.socket.onopen = () => {
      this.startHeartbeat()
      this.handlers.onOpen?.()
    }
    this.socket.onclose = () => {
      this.stopHeartbeat()
      this.handlers.onClose?.()
      this.socket = null
      if (!this.intentionalClose) {
        this.reconnectTimer = window.setTimeout(() => this.connect(), 2000)
      }
    }
    this.socket.onerror = () => {
      this.handlers.onError?.('实时连接异常')
    }
    this.socket.onmessage = (event) => {
      const payload = JSON.parse(event.data) as Envelope
      this.handlers.onMessage?.(payload)
    }
  }

  disconnect() {
    this.intentionalClose = true
    this.stopHeartbeat()
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer)
    }
    this.socket?.close()
    this.socket = null
  }

  send(type: string, payload?: unknown) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return
    }
    this.socket.send(JSON.stringify({ type, payload }))
  }

  private startHeartbeat() {
    this.stopHeartbeat()
    this.heartbeatTimer = window.setInterval(() => {
      this.send('ping')
    }, 15000)
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      window.clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }
}
