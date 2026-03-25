import { defineStore } from 'pinia'

import { getMe, guestLogin } from '../api/auth'
import type { User } from '../types'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('hz_token') ?? '',
    user: null as User | null,
    loading: false
  }),
  actions: {
    async restore() {
      if (!this.token) {
        return
      }
      if (this.user) {
        return
      }
      try {
        const response = await getMe()
        this.user = response.user
      } catch {
        this.logout()
      }
    },
    async login(username: string) {
      this.loading = true
      try {
        const response = await guestLogin(username)
        this.token = response.token
        this.user = response.user
        localStorage.setItem('hz_token', response.token)
      } finally {
        this.loading = false
      }
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem('hz_token')
    }
  }
})
