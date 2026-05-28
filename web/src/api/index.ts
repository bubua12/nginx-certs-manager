/**
 * Axios 实例配置文件 (api/index.ts)
 * 创建并配置全局 HTTP 客户端实例，统一处理请求和响应拦截
 */

// 导入 Axios HTTP 客户端库
import axios from 'axios'
// 导入路由实例，用于 401 时跳转到登录页
import router from '@/router'

// 创建 Axios 实例，设置基础配置
const api = axios.create({
  // 所有请求的基础 URL 前缀，会与各接口的相对路径拼接
  // 例如 '/auth/login' 实际请求为 '/api/auth/login'
  baseURL: '/api',
  // 请求超时时间：30 秒
  timeout: 30000,
})

/**
 * 请求拦截器
 * 在每个 HTTP 请求发出前自动执行
 * 作用：自动将本地存储的 token 附加到请求头中，实现身份认证
 */
api.interceptors.request.use((config) => {
  // 从 localStorage 读取之前登录时保存的 JWT token
  const token = localStorage.getItem('token')
  if (token) {
    // 将 token 以 Bearer 格式添加到 Authorization 请求头
    config.headers.Authorization = `Bearer ${token}`
  }
  // 返回修改后的配置，继续发送请求
  return config
})

/**
 * 响应拦截器
 * 在每个 HTTP 响应返回后自动执行
 * 作用：统一处理 401 未授权错误（token 过期或无效）
 */
api.interceptors.response.use(
  // 成功响应：直接返回，不做处理
  (res) => res,
  // 错误响应处理
  (err) => {
    // 如果返回 401 状态码，说明认证失败
    if (err.response?.status === 401) {
      // 清除本地存储的认证信息（token、用户名、角色）
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      localStorage.removeItem('role')
      // 重定向到登录页面
      router.push('/login')
    }
    // 将错误继续抛出，让调用方可以捕获处理
    return Promise.reject(err)
  }
)

// 导出配置好的 Axios 实例，供其他模块使用
export default api
