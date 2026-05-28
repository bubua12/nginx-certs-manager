<!-- 登录页面模板 -->
<template>
  <!-- 登录页最外层容器，应用渐变背景 -->
  <div class="login-container">
    <!-- 登录卡片，包含标题和表单 -->
    <div class="login-card">
      <!-- 登录页头部：显示锁图标、应用名称和提示文字 -->
      <div class="login-header">
        <el-icon :size="40" color="#409eff"><Lock /></el-icon>
        <h1>Nginx Certs Manager</h1>
        <p>请登录以继续</p>
      </div>

      <!-- 登录表单 -->
      <!-- ref: 获取表单组件引用，用于手动触发表单验证 -->
      <!-- model: 绑定表单数据对象 -->
      <!-- rules: 绑定验证规则 -->
      <!-- @submit.prevent: 阻止表单默认提交行为，改为调用 handleLogin -->
      <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin" label-position="top">
        <!-- 用户名输入框 -->
        <el-form-item label="用户名" prop="username">
          <!-- prefix-icon: 输入框左侧图标；@keyup.enter: 按回车键触发登录 -->
          <el-input v-model="form.username" placeholder="请输入用户名" size="large" prefix-icon="User" @keyup.enter="handleLogin" />
        </el-form-item>
        <!-- 密码输入框 -->
        <el-form-item label="密码" prop="password">
          <!-- show-password: 显示密码可见性切换按钮 -->
          <el-input v-model="form.password" type="password" placeholder="请输入密码" size="large" prefix-icon="Lock" show-password @keyup.enter="handleLogin" />
        </el-form-item>
        <!-- 登录按钮 -->
        <el-form-item>
          <!-- :loading: 登录请求中时显示加载动画，防止重复提交 -->
          <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="handleLogin">
            登 录
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * 登录页面逻辑 (Login.vue)
 * 功能：用户输入用户名和密码进行身份认证
 * 登录成功后跳转到仪表盘页面，失败时显示错误信息（含剩余尝试次数）
 */

// Vue 3 响应式 API：ref 用于基本类型，reactive 用于对象
import { ref, reactive } from 'vue'
// 路由实例，用于登录成功后的页面跳转
import { useRouter } from 'vue-router'
// Element Plus 消息提示组件
import { ElMessage } from 'element-plus'
// 认证状态管理 Store，提供 login 方法
import { useAuthStore } from '@/stores/auth'

// 获取路由实例
const router = useRouter()
// 获取认证 Store 实例
const authStore = useAuthStore()
// 表单组件引用，用于调用表单的 validate() 方法进行验证
const formRef = ref()
// 登录按钮的加载状态，防止重复提交
const loading = ref(false)
// 表单数据（响应式对象），双向绑定到输入框
const form = reactive({ username: '', password: '' })

// 表单验证规则
const rules = {
  // 用户名：必填，失去焦点时触发验证
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  // 密码：必填，失去焦点时触发验证
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

/**
 * 处理登录操作
 * 1. 先验证表单必填项
 * 2. 调用 authStore.login() 发送登录请求
 * 3. 成功后跳转到仪表盘
 * 4. 失败时显示错误信息，包含后端返回的剩余尝试次数
 */
const handleLogin = async () => {
  // 触发表单验证，如果验证不通过会抛出异常
  await formRef.value?.validate()
  // 设置按钮为加载状态
  loading.value = true
  try {
    // 调用认证 Store 的登录方法
    await authStore.login(form.username, form.password)
    // 登录成功提示
    ElMessage.success('登录成功')
    // 跳转到仪表盘页面
    router.push('/dashboard')
  } catch (e: any) {
    // 登录失败：获取后端返回的错误信息
    const msg = e.response?.data?.error || '登录失败'
    // 获取剩余登录尝试次数（后端有登录失败次数限制）
    const remaining = e.response?.data?.remaining
    if (remaining !== undefined && remaining > 0) {
      // 如果还有剩余次数，显示剩余次数提示
      ElMessage.error(`${msg}，剩余 ${remaining} 次机会`)
    } else {
      // 否则直接显示错误信息
      ElMessage.error(msg)
    }
  } finally {
    // 无论成功失败，都取消加载状态
    loading.value = false
  }
}
</script>

<!-- 登录页面样式 -->
<style scoped>
/* 登录页最外层容器：全屏居中，渐变紫色背景 */
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}
/* 登录卡片：白色圆角卡片，带阴影效果 */
.login-card {
  width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
}
/* 登录头部区域：居中显示图标和文字 */
.login-header {
  text-align: center;
  margin-bottom: 32px;
}
/* 应用标题样式 */
.login-header h1 { font-size: 22px; margin: 12px 0 4px; color: #303133; }
/* 提示文字样式 */
.login-header p { color: #909399; font-size: 14px; }
</style>
