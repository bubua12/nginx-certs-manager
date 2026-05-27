<template>
  <router-view v-if="isLoginPage" />
  <el-container v-else class="app-container">
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
        <el-icon><Fold v-if="!isCollapsed" /><Expand v-else /></el-icon>
      </div>
    </el-aside>
    <el-container>
      <el-header class="app-header">
        <h2>{{ currentTitle }}</h2>
        <div class="header-right">
          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-icon><User /></el-icon>
              {{ authStore.username || 'admin' }}
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="password">修改密码</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>

  <!-- 修改密码弹窗 -->
  <el-dialog v-model="showPasswordDialog" title="修改密码" width="400" :close-on-click-modal="false">
    <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="80px">
      <el-form-item label="原密码" prop="old_password">
        <el-input v-model="pwdForm.old_password" type="password" show-password />
      </el-form-item>
      <el-form-item label="新密码" prop="new_password">
        <el-input v-model="pwdForm.new_password" type="password" show-password placeholder="至少6位" />
      </el-form-item>
      <el-form-item label="确认密码" prop="confirm_password">
        <el-input v-model="pwdForm.confirm_password" type="password" show-password />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showPasswordDialog = false">取消</el-button>
      <el-button type="primary" @click="handleChangePassword" :loading="pwdLoading">确认修改</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import api from '@/api/index'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const isCollapsed = ref(false)

const isLoginPage = computed(() => route.path === '/login')

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

// 修改密码
const showPasswordDialog = ref(false)
const pwdLoading = ref(false)
const pwdFormRef = ref()
const pwdForm = reactive({ old_password: '', new_password: '', confirm_password: '' })
const pwdRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' },
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    {
      validator: (_r: any, value: string, callback: Function) => {
        if (value !== pwdForm.new_password) callback(new Error('两次密码不一致'))
        else callback()
      },
      trigger: 'blur',
    },
  ],
}

const handleChangePassword = async () => {
  await pwdFormRef.value?.validate()
  pwdLoading.value = true
  try {
    await api.post('/auth/change-password', {
      old_password: pwdForm.old_password,
      new_password: pwdForm.new_password,
    })
    ElMessage.success('密码修改成功，请重新登录')
    showPasswordDialog.value = false
    pwdForm.old_password = ''
    pwdForm.new_password = ''
    pwdForm.confirm_password = ''
    authStore.logout()
    router.push('/login')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '修改失败')
  } finally {
    pwdLoading.value = false
  }
}

const handleCommand = async (cmd: string) => {
  if (cmd === 'logout') {
    await ElMessageBox.confirm('确定退出登录？', '提示', { type: 'warning' })
    authStore.logout()
    router.push('/login')
  } else if (cmd === 'password') {
    showPasswordDialog.value = true
  }
}
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
.app-menu { flex: 1; border-right: none !important; overflow-y: auto; }
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
  justify-content: space-between;
  border-bottom: 1px solid #e8e8e8;
  background: #fff;
  padding: 0 20px;
}
.app-header h2 { font-size: 18px; font-weight: 500; }
.header-right { display: flex; align-items: center; }
.user-info {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  color: #606266;
  font-size: 14px;
}
.user-info:hover { color: #409eff; }
.app-main { background: #f0f2f5; overflow-y: auto; }
</style>
