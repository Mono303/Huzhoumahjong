import { request } from './http'
import type { User } from '../types'

export function guestLogin(username: string) {
  return request<{ token: string; user: User }>('/auth/guest', {
    method: 'POST',
    body: JSON.stringify({ username })
  })
}

export function getMe() {
  return request<{ user: User }>('/users/me')
}
