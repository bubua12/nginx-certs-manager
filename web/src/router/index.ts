import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { title: '登录', public: true },
    },
    {
      path: '/dashboard',
      name: 'Dashboard',
      component: () => import('@/views/Dashboard.vue'),
      meta: { title: '仪表盘', icon: 'DataBoard' },
    },
    {
      path: '/certificates',
      name: 'Certificates',
      component: () => import('@/views/CertificateList.vue'),
      meta: { title: '证书管理', icon: 'Lock' },
    },
    {
      path: '/certificates/:id',
      name: 'CertificateDetail',
      component: () => import('@/views/CertificateDetail.vue'),
      meta: { title: '证书详情', hidden: true },
    },
    {
      path: '/sites',
      name: 'Sites',
      component: () => import('@/views/SiteList.vue'),
      meta: { title: '站点管理', icon: 'Monitor' },
    },
    {
      path: '/sites/:id',
      name: 'SiteDetail',
      component: () => import('@/views/SiteDetail.vue'),
      meta: { title: '站点详情', hidden: true },
    },
    {
      path: '/settings',
      name: 'Settings',
      component: () => import('@/views/Settings.vue'),
      meta: { title: '系统设置', icon: 'Setting' },
    },
  ],
})

// Route guard - redirect to login if not authenticated
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.public) {
    next()
  } else if (!token) {
    next('/login')
  } else {
    next()
  }
})

export default router
