import { createRouter, createWebHistory } from 'vue-router'

import LoginView from '../views/LoginView.vue'
import LobbyView from '../views/LobbyView.vue'
import RoomView from '../views/RoomView.vue'
import GamePlayableView from '../views/GamePlayableView.vue'

const authRequired = (toPath: string) => toPath !== '/login' && !localStorage.getItem('hz_token')

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/lobby' },
    { path: '/login', component: LoginView },
    { path: '/lobby', component: LobbyView },
    { path: '/room/:code', component: RoomView, props: true },
    { path: '/game/:code', component: GamePlayableView, props: true }
  ]
})

router.beforeEach((to) => {
  if (authRequired(to.path)) {
    return '/login'
  }
  if (to.path === '/login' && localStorage.getItem('hz_token')) {
    return '/lobby'
  }
  return true
})
