<template>
  <el-container class="app-container">
    <el-aside :width="isCollapsed ? '64px' : '220px'" class="app-aside">
      <div class="logo" @click="router.push('/')">
        <el-icon :size="24"><Lock /></el-icon>
        <span v-show="!isCollapsed" class="logo-text">Certs Manager</span>
      </div>
      <el-menu
        :default-active="route.path"
        :collapse="isCollapsed"
        router
        background-color="#001529"
        text-color="#ffffffa6"
        active-text-color="#409eff"
        class="app-menu"
      >
        <el-menu-item
          v-for="item in menuItems"
          :key="item.path"
          :index="item.path"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.title }}</template>
        </el-menu-item>
      </el-menu>
      <div class="collapse-btn" @click="isCollapsed = !isCollapsed">
        <el-icon>
          <Fold v-if="!isCollapsed" />
          <Expand v-else />
        </el-icon>
      </div>
    </el-aside>
    <el-container>
      <el-header class="app-header">
        <h2>{{ currentTitle }}</h2>
      </el-header>
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const isCollapsed = ref(false)

const menuItems = [
  { path: '/dashboard', title: '仪表盘', icon: 'DataBoard' },
  { path: '/certificates', title: '证书管理', icon: 'Lock' },
  { path: '/sites', title: '站点管理', icon: 'Monitor' },
  { path: '/settings', title: '系统设置', icon: 'Setting' },
]

const currentTitle = computed(() => {
  const item = menuItems.find((m) => m.path === route.path)
  return item?.title || 'Nginx Certs Manager'
})
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, #app { height: 100%; }
.app-container { height: 100vh; }
.app-aside {
  background: #001529;
  transition: width 0.3s;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #fff;
  font-size: 18px;
  font-weight: bold;
  cursor: pointer;
  border-bottom: 1px solid #ffffff1a;
  flex-shrink: 0;
}
.logo-text { white-space: nowrap; }
.app-menu {
  flex: 1;
  border-right: none !important;
  overflow-y: auto;
}
.app-menu::-webkit-scrollbar { width: 0; }
.collapse-btn {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffffa6;
  cursor: pointer;
  border-top: 1px solid #ffffff1a;
  flex-shrink: 0;
}
.collapse-btn:hover { color: #fff; }
.app-header {
  height: 60px;
  display: flex;
  align-items: center;
  border-bottom: 1px solid #e8e8e8;
  background: #fff;
}
.app-header h2 { font-size: 18px; font-weight: 500; }
.app-main {
  background: #f0f2f5;
  overflow-y: auto;
}
</style>
