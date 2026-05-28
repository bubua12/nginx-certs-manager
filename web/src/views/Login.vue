<!-- 登录页面模板 -->
<template>
  <!-- 登录页最外层容器 -->
  <div class="login-container">
    <!-- 背景装饰：动态渐变圆形 -->
    <div class="bg-decoration">
      <div class="circle circle-1"></div>
      <div class="circle circle-2"></div>
      <div class="circle circle-3"></div>
    </div>

    <!-- 登录卡片：毛玻璃效果 -->
    <div class="login-card">
      <!-- 登录页头部：锁图标 + 应用名称 + 副标题 -->
      <div class="login-header">
        <div class="logo-icon">
          <el-icon :size="36" color="#fff"><Lock /></el-icon>
        </div>
        <h1>Nginx Certs Manager</h1>
        <p>SSL 证书 & Nginx 站点管理平台</p>
      </div>

      <!-- 登录表单 -->
      <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin" label-position="top">
        <!-- 用户名输入框 -->
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" size="large" prefix-icon="User" @keyup.enter="handleLogin" />
        </el-form-item>
        <!-- 密码输入框 -->
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" size="large" prefix-icon="Lock" show-password @keyup.enter="handleLogin" />
        </el-form-item>
        <!-- 登录按钮 -->
        <el-form-item>
          <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="handleLogin" class="login-btn">
            登 录
          </el-button>
        </el-form-item>
      </el-form>

      <!-- 底部信息 -->
      <div class="login-footer">
        <span>Let's Encrypt + Certbot + Nginx</span>
      </div>
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
  await formRef.value?.validate()
  loading.value = true
  try {
    await authStore.login(form.username, form.password)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } catch (e: any) {
    const msg = e.response?.data?.error || '登录失败'
    const remaining = e.response?.data?.remaining
    if (remaining !== undefined && remaining > 0) {
      ElMessage.error(`${msg}，剩余 ${remaining} 次机会`)
    } else {
      ElMessage.error(msg)
    }
  } finally {
    loading.value = false
  }
}
</script>

<!-- 登录页面样式 -->
<style scoped>
/* 登录页最外层容器：全屏，深色渐变背景 */
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0c1426 0%, #1a1a3e 40%, #16213e 100%);
  position: relative;
  overflow: hidden;
}

/* ===== 背景装饰：浮动渐变圆形 ===== */
.bg-decoration {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}
.circle {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.4;
}
/* 右上角蓝紫色圆形 */
.circle-1 {
  width: 500px;
  height: 500px;
  background: linear-gradient(135deg, #667eea, #764ba2);
  top: -150px;
  right: -100px;
  animation: float1 15s ease-in-out infinite;
}
/* 左下角蓝绿色圆形 */
.circle-2 {
  width: 400px;
  height: 400px;
  background: linear-gradient(135deg, #43e97b, #38f9d7);
  bottom: -100px;
  left: -80px;
  animation: float2 18s ease-in-out infinite;
}
/* 中间偏右橙粉色圆形 */
.circle-3 {
  width: 300px;
  height: 300px;
  background: linear-gradient(135deg, #fa709a, #fee140);
  top: 40%;
  right: 20%;
  animation: float3 20s ease-in-out infinite;
}

/* 圆形浮动动画 */
@keyframes float1 {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(-40px, 40px) scale(1.1); }
}
@keyframes float2 {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(30px, -30px) scale(1.15); }
}
@keyframes float3 {
  0%, 100% { transform: translate(0, 0) scale(1); }
  50% { transform: translate(-20px, -40px) scale(0.9); }
}

/* ===== 登录卡片：毛玻璃效果 ===== */
.login-card {
  width: 420px;
  padding: 48px 40px 32px;
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  box-shadow: 0 32px 64px rgba(0, 0, 0, 0.3);
  position: relative;
  z-index: 1;
}

/* 登录头部区域 */
.login-header {
  text-align: center;
  margin-bottom: 36px;
}
/* Logo 图标容器：蓝紫渐变圆形背景 */
.logo-icon {
  width: 68px;
  height: 68px;
  margin: 0 auto 16px;
  background: linear-gradient(135deg, #667eea, #764ba2);
  border-radius: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 24px rgba(102, 126, 234, 0.4);
}
/* 应用标题 */
.login-header h1 {
  font-size: 24px;
  font-weight: 700;
  margin: 0 0 6px;
  color: #fff;
  letter-spacing: 0.5px;
}
/* 副标题 */
.login-header p {
  color: rgba(255, 255, 255, 0.5);
  font-size: 14px;
  margin: 0;
}

/* ===== 表单样式覆盖：适配深色背景 ===== */
/* 表单标签颜色 */
.login-card :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.7);
  font-size: 13px;
}
/* 输入框容器 */
.login-card :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
  border-radius: 10px;
  height: 44px;
  transition: all 0.3s;
}
/* 输入框聚焦状态 */
.login-card :deep(.el-input__wrapper:hover),
.login-card :deep(.el-input__wrapper.is-focus) {
  border-color: #667eea;
  background: rgba(255, 255, 255, 0.1);
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.15);
}
/* 输入框文字颜色 */
.login-card :deep(.el-input__inner) {
  color: #fff;
}
/* 输入框占位符颜色 */
.login-card :deep(.el-input__inner::placeholder) {
  color: rgba(255, 255, 255, 0.3);
}
/* 输入框图标颜色 */
.login-card :deep(.el-input__prefix .el-icon) {
  color: rgba(255, 255, 255, 0.4);
}

/* 登录按钮 */
.login-btn {
  height: 46px;
  border-radius: 10px;
  font-size: 16px;
  font-weight: 600;
  background: linear-gradient(135deg, #667eea, #764ba2) !important;
  border: none !important;
  box-shadow: 0 8px 24px rgba(102, 126, 234, 0.35);
  transition: all 0.3s;
}
.login-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 32px rgba(102, 126, 234, 0.5);
}

/* 底部信息 */
.login-footer {
  text-align: center;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}
.login-footer span {
  color: rgba(255, 255, 255, 0.25);
  font-size: 12px;
  letter-spacing: 1px;
}
</style>
