/**
 * 认证状态管理 Store (stores/auth.ts)
 * 使用 Pinia 管理用户登录状态、token、用户信息
 * 采用 Composition API 风格（setup 语法）
 */

// 从 Pinia 导入 defineStore，用于创建状态管理仓库
import { defineStore } from 'pinia'
// 导入 Vue 响应式 API
import { ref } from 'vue'
// 导入 Axios 实例，用于调用后端认证接口
import api from '@/api/index'

/**
 * 定义认证状态管理 Store
 * 'auth' 是该 store 的唯一标识符
 * 使用 setup 函数模式（类似 Vue 3 的 setup 语法）
 */
export const useAuthStore = defineStore('auth', () => {
  // ==================== 响应式状态 ====================

  // 认证令牌（JWT token），初始化时从 localStorage 恢复，保持登录状态
  const token = ref(localStorage.getItem('token') || '')
  // 当前登录用户名
  const username = ref(localStorage.getItem('username') || '')
  // 当前用户角色（如 admin）
  const role = ref(localStorage.getItem('role') || '')
  // 是否已登录的标志位，根据 token 是否存在来判断
  const isLoggedIn = ref(!!token.value)

  // ==================== 操作方法 ====================

  /**
   * 用户登录
   * @param user - 用户名
   * @param password - 密码
   * 调用后端 /api/auth/login 接口，成功后将 token 和用户信息保存到 store 和 localStorage
   */
  async function login(user: string, password: string) {
    // 发送登录请求
    const res = await api.post('/auth/login', { username: user, password })
    const data = res.data

    // 更新 store 中的响应式状态
    token.value = data.token
    username.value = data.username
    role.value = data.role
    isLoggedIn.value = true

    // 将认证信息持久化到 localStorage，刷新页面后可恢复登录状态
    localStorage.setItem('token', data.token)
    localStorage.setItem('username', data.username)
    localStorage.setItem('role', data.role)
  }

  /**
   * 用户登出
   * 清除所有认证信息（store 状态 + localStorage）
   * 调用后会跳转到登录页面（由 App.vue 中的 handleCommand 触发）
   */
  function logout() {
    // 清空 store 中的状态
    token.value = ''
    username.value = ''
    role.value = ''
    isLoggedIn.value = false

    // 清除 localStorage 中的认证信息
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('role')
  }

  /**
   * 获取当前用户信息
   * 调用后端 /api/auth/me 接口验证 token 有效性并刷新用户信息
   * 如果 token 已失效（接口返回错误），自动调用 logout 清除状态
   */
  async function fetchUser() {
    try {
      const res = await api.get('/auth/me')
      // 更新用户名和角色信息
      username.value = res.data.username
      role.value = res.data.role
      // 同步更新 localStorage
      localStorage.setItem('username', res.data.username)
      localStorage.setItem('role', res.data.role)
    } catch {
      // 如果获取用户信息失败（token 过期等），执行登出操作
      logout()
    }
  }

  // 返回所有状态和方法，供组件通过 useAuthStore() 访问
  return { token, username, role, isLoggedIn, login, logout, fetchUser }
})
