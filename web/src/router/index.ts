/**
 * 路由配置文件 (router/index.ts)
 * 定义应用的所有页面路由、路由守卫（权限控制）
 */

// 从 Vue Router 导入创建路由实例和使用 HTML5 History 模式的方法
import { createRouter, createWebHistory } from 'vue-router'

// 创建路由实例
const router = createRouter({
  // 使用 HTML5 History 模式（URL 不带 # 号，需要后端配合）
  history: createWebHistory(),
  // 路由表定义
  routes: [
    // 根路径重定向到仪表盘页面
    { path: '/', redirect: '/dashboard' },
    // 登录页 - 公开页面，无需认证
    {
      path: '/login',
      name: 'Login',
      // 使用动态 import 实现路由懒加载，减少首屏加载体积
      component: () => import('@/views/Login.vue'),
      // meta: 路由元信息
      // title: 页面标题，public: true 表示该页面不需要登录即可访问
      meta: { title: '登录', public: true },
    },
    // 仪表盘页面 - 展示系统概览和统计数据
    {
      path: '/dashboard',
      name: 'Dashboard',
      component: () => import('@/views/Dashboard.vue'),
      meta: { title: '仪表盘', icon: 'DataBoard' },
    },
    // 证书列表页面 - 展示所有 SSL 证书及管理操作
    {
      path: '/certificates',
      name: 'Certificates',
      component: () => import('@/views/CertificateList.vue'),
      meta: { title: '证书管理', icon: 'Lock' },
    },
    // 证书详情页面 - 展示单个证书的详细信息（隐藏在侧边栏菜单中）
    {
      path: '/certificates/:id',
      name: 'CertificateDetail',
      component: () => import('@/views/CertificateDetail.vue'),
      // hidden: true 表示该路由不出现在侧边栏导航菜单中
      meta: { title: '证书详情', hidden: true },
    },
    // 站点列表页面 - 展示所有 Nginx 站点及管理操作
    {
      path: '/sites',
      name: 'Sites',
      component: () => import('@/views/SiteList.vue'),
      meta: { title: '站点管理', icon: 'Monitor' },
    },
    // 站点详情页面 - 展示单个站点的配置和管理操作（隐藏在侧边栏菜单中）
    {
      path: '/sites/:id',
      name: 'SiteDetail',
      component: () => import('@/views/SiteDetail.vue'),
      meta: { title: '站点详情', hidden: true },
    },
    // 系统设置页面 - Nginx 状态查看、系统参数配置、操作日志
    {
      path: '/settings',
      name: 'Settings',
      component: () => import('@/views/Settings.vue'),
      meta: { title: '系统设置', icon: 'Setting' },
    },
  ],
})

/**
 * 全局前置路由守卫
 * 在每次路由跳转前执行，用于权限验证
 * @param to - 即将进入的目标路由
 * @param _from - 当前正要离开的路由（未使用）
 * @param next - 放行函数，调用后才会真正跳转
 */
router.beforeEach((to, _from, next) => {
  // 从本地存储获取认证 token
  const token = localStorage.getItem('token')
  // 如果目标页面是公开页面（如登录页），直接放行
  if (to.meta.public) {
    next()
  } else if (!token) {
    // 如果没有 token，说明未登录，重定向到登录页
    next('/login')
  } else {
    // 已登录，正常放行
    next()
  }
})

// 导出路由实例，供 main.ts 注册使用
export default router
