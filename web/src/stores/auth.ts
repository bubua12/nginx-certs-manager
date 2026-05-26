import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api/index'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref(localStorage.getItem('username') || '')
  const role = ref(localStorage.getItem('role') || '')

  const isLoggedIn = ref(!!token.value)

  async function login(user: string, password: string) {
    const res = await api.post('/auth/login', { username: user, password })
    const data = res.data

    token.value = data.token
    username.value = data.username
    role.value = data.role
    isLoggedIn.value = true

    localStorage.setItem('token', data.token)
    localStorage.setItem('username', data.username)
    localStorage.setItem('role', data.role)
  }

  function logout() {
    token.value = ''
    username.value = ''
    role.value = ''
    isLoggedIn.value = false

    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('role')
  }

  async function fetchUser() {
    try {
      const res = await api.get('/auth/me')
      username.value = res.data.username
      role.value = res.data.role
      localStorage.setItem('username', res.data.username)
      localStorage.setItem('role', res.data.role)
    } catch {
      logout()
    }
  }

  return { token, username, role, isLoggedIn, login, logout, fetchUser }
})
