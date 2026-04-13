<template>
  <main class="shell shell-center">
    <section class="panel auth-panel">
      <p class="eyebrow">Huzhou Mahjong Online</p>
      <h1>湖州麻将</h1>
      <p class="subtle">从单文件玩具页升级成可联机、可维护的网页麻将项目。</p>

      <form class="stack" @submit.prevent="submit">
        <label class="field">
          <span>游客昵称</span>
          <input v-model.trim="username" maxlength="12" placeholder="输入你的名字" />
        </label>
        <button class="primary-button" :disabled="auth.loading">进入大厅</button>
      </form>

      <p v-if="error" class="error-text">{{ error }}</p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const username = ref('玩家')
const error = ref('')

async function submit() {
  error.value = ''
  try {
    await auth.login(username.value)
    await router.replace('/lobby')
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录失败'
  }
}
</script>
