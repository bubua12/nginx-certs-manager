<!--
  证书详情页面模板 (CertificateDetail.vue)
  功能：展示单个证书的完整信息，提供续期和撤销操作，以及到期倒计时可视化
-->
<template>
  <!-- v-loading: 页面加载时显示遮罩动画 -->
  <div v-loading="loading">
    <!-- 页面头部导航：显示返回按钮和证书域名 -->
    <el-page-header @back="$router.push('/certificates')" :content="cert?.domain || '证书详情'" style="margin-bottom: 20px" />

    <el-row :gutter="20">
      <!-- 左侧：证书详细信息卡片（占比 2/3） -->
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header><span>证书信息</span></template>
          <!-- 使用 Descriptions 组件展示证书信息，两列布局，带边框 -->
          <el-descriptions :column="2" border>
            <!-- 域名 -->
            <el-descriptions-item label="域名">{{ cert?.domain }}</el-descriptions-item>
            <!-- 状态标签：正常(绿)/即将过期(橙)/已过期(红) -->
            <el-descriptions-item label="状态">
              <el-tag :type="statusType(cert?.status)">{{ statusLabel(cert?.status) }}</el-tag>
            </el-descriptions-item>
            <!-- 签发机构（如 Let's Encrypt） -->
            <el-descriptions-item label="签发机构">{{ cert?.issuer || '-' }}</el-descriptions-item>
            <!-- 是否开启自动续期 -->
            <el-descriptions-item label="自动续期">
              <el-tag :type="cert?.auto_renew ? 'success' : 'info'" size="small">
                {{ cert?.auto_renew ? '是' : '否' }}
              </el-tag>
            </el-descriptions-item>
            <!-- 证书生效时间 -->
            <el-descriptions-item label="生效时间">{{ formatDate(cert?.not_before) }}</el-descriptions-item>
            <!-- 证书过期时间 -->
            <el-descriptions-item label="过期时间">{{ formatDate(cert?.not_after) }}</el-descriptions-item>
            <!-- 剩余天数：30 天内显示红色 -->
            <el-descriptions-item label="剩余天数">
              <span :style="{ color: (cert?.days_left || 0) <= 30 ? '#f56c6c' : '#67c23a', fontWeight: 'bold', fontSize: '16px' }">
                {{ cert?.days_left ?? '-' }} 天
              </span>
            </el-descriptions-item>
            <!-- 上次续期时间 -->
            <el-descriptions-item label="上次续期">{{ formatDate(cert?.last_renewed) || '-' }}</el-descriptions-item>
            <!-- 证书文件路径（占两列宽度） -->
            <el-descriptions-item label="证书路径" :span="2">{{ cert?.cert_path || '-' }}</el-descriptions-item>
            <!-- 私钥文件路径（占两列宽度） -->
            <el-descriptions-item label="私钥路径" :span="2">{{ cert?.key_path || '-' }}</el-descriptions-item>
            <!-- SAN（Subject Alternative Name）域名列表（占两列宽度） -->
            <!-- 遍历 parsedSANs 数组，每个域名显示为一个小标签 -->
            <el-descriptions-item label="SAN 域名" :span="2">
              <el-tag v-for="san in parsedSANs" :key="san" style="margin: 2px 4px" size="small">{{ san }}</el-tag>
              <span v-if="!parsedSANs.length">-</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <!-- 右侧：操作和到期倒计时卡片（占比 1/3） -->
      <el-col :span="8">
        <!-- 操作卡片：续期和撤销按钮 -->
        <el-card shadow="hover">
          <template #header><span>操作</span></template>
          <div style="display: flex; flex-direction: column; gap: 12px">
            <!-- 立即续期按钮 -->
            <el-button type="primary" size="large" @click="handleRenew" :loading="renewing">
              <el-icon><Refresh /></el-icon>立即续期
            </el-button>
            <!-- 撤销证书按钮（危险操作） -->
            <el-button type="danger" size="large" @click="handleRevoke">
              <el-icon><Delete /></el-icon>撤销证书
            </el-button>
          </div>
        </el-card>

        <!-- 到期倒计时卡片 -->
        <el-card shadow="hover" style="margin-top: 20px">
          <template #header><span>到期倒计时</span></template>
          <!-- 大号数字显示剩余天数 -->
          <div class="countdown">
            <div class="countdown-number" :style="{ color: countdownColor }">
              {{ cert?.days_left ?? '-' }}
            </div>
            <div class="countdown-label">天</div>
          </div>
          <!-- 进度条：直观展示证书有效期使用情况 -->
          <!-- :percentage: 进度百分比（按 90 天证书周期计算） -->
          <!-- :color: 进度条颜色根据剩余天数变化 -->
          <el-progress
            :percentage="progressPercent"
            :color="progressColor"
            :stroke-width="12"
            style="margin-top: 16px"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
/**
 * 证书详情页面逻辑 (CertificateDetail.vue)
 * 功能：
 * 1. 加载并展示单个证书的完整信息（域名、状态、时间、路径、SAN 等）
 * 2. 提供证书续期和撤销操作
 * 3. 通过大号数字和进度条直观展示到期倒计时
 */

// Vue 响应式 API
import { ref, computed, onMounted } from 'vue'
// 路由相关：useRoute 获取路由参数（证书 ID），useRouter 用于跳转
import { useRoute, useRouter } from 'vue-router'
// Element Plus 消息提示和确认弹窗
import { ElMessage, ElMessageBox } from 'element-plus'
// 导入证书相关 API 接口
import { getCertificate, renewCertificate, revokeCertificate } from '@/api/modules'
// 日期格式化工具
import dayjs from 'dayjs'

// 获取当前路由信息（用于读取 URL 中的证书 ID 参数）
const route = useRoute()
// 获取路由实例（用于跳转回证书列表页）
const router = useRouter()
// 证书详情数据对象
const cert = ref<any>(null)
// 页面加载状态
const loading = ref(false)
// 续期按钮的加载状态
const renewing = ref(false)

// ==================== 工具函数 ====================

/**
 * 格式化日期为 "YYYY-MM-DD HH:mm" 格式
 * @param d - ISO 格式的日期字符串
 * @returns 格式化后的日期字符串，无效日期返回 '-'
 */
const formatDate = (d: string) => {
  if (!d) return '-'
  const date = new Date(d)
  // 年份 > 2000 才认为是有效日期（防止显示 1970 年等异常值）
  return date.getFullYear() > 2000 ? dayjs(d).format('YYYY-MM-DD HH:mm') : '-'
}

// 将证书状态转换为 Element Plus Tag 的颜色类型
const statusType = (s: string) => ({ active: 'success', expiring: 'warning', expired: 'danger' }[s] || 'info') as any

// 将证书状态转换为中文显示
const statusLabel = (s: string) => ({ active: '正常', expiring: '即将过期', expired: '已过期' }[s] || s || '-')

// ==================== 计算属性 ====================

/**
 * 解析 SAN（Subject Alternative Name）域名列表
 * 后端存储为 JSON 字符串，解析为数组供模板遍历
 */
const parsedSANs = computed(() => {
  try { return cert.value?.sans ? JSON.parse(cert.value.sans) : [] } catch { return [] }
})

/**
 * 倒计时数字颜色
 * - 已过期（天数 < 0）：红色
 * - 即将过期（天数 <= 30）：橙色
 * - 正常：绿色
 */
const countdownColor = computed(() => {
  const d = cert.value?.days_left ?? 0
  return d < 0 ? '#f56c6c' : d <= 30 ? '#e6a23c' : '#67c23a'
})

/**
 * 到期进度条百分比
 * 按 90 天证书周期计算（大多数免费证书有效期为 90 天）
 * 限制在 0-100 范围内
 */
const progressPercent = computed(() => {
  const d = cert.value?.days_left ?? 0
  return Math.round(Math.max(0, Math.min(100, (d / 90) * 100)))
})

/**
 * 进度条颜色（与倒计时颜色逻辑一致）
 */
const progressColor = computed(() => {
  const d = cert.value?.days_left ?? 0
  return d < 0 ? '#f56c6c' : d <= 30 ? '#e6a23c' : '#67c23a'
})

// ==================== 数据加载和操作函数 ====================

/**
 * 加载证书详情数据
 * 从 URL 参数中获取证书 ID，调用 API 获取详情，并手动计算剩余天数
 */
const loadCert = async () => {
  loading.value = true
  try {
    // 从路由参数获取证书 ID
    const res = await getCertificate(Number(route.params.id))
    const d = res.data
    // 手动计算证书剩余天数（后端可能不返回该字段）
    const expiry = new Date(d.not_after)
    const daysLeft = expiry.getFullYear() > 2000
      ? Math.floor((expiry.getTime() - Date.now()) / 86400000) // 毫秒转天数
      : -99999  // 无效日期视为已过期
    // 将剩余天数附加到证书数据中
    cert.value = { ...d, days_left: daysLeft }
  } catch {
    ElMessage.error('加载证书信息失败')
  } finally {
    loading.value = false
  }
}

/**
 * 处理证书续期操作
 * 1. 确认对话框
 * 2. 调用续期 API
 * 3. 成功后重新加载证书详情（刷新数据）
 */
const handleRenew = async () => {
  try {
    await ElMessageBox.confirm(`确定要续期证书 ${cert.value.domain} 吗？`, '确认续期', { type: 'warning' })
    renewing.value = true
    await renewCertificate(cert.value.id)
    ElMessage.success('续期成功')
    // 重新加载证书信息以获取最新状态
    loadCert()
  } catch (e: any) {
    // 忽略用户取消操作，其他错误显示提示
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '续期失败')
  } finally {
    renewing.value = false
  }
}

/**
 * 处理证书撤销操作
 * 1. 危险确认对话框（强调不可恢复）
 * 2. 调用撤销 API
 * 3. 成功后跳转回证书列表页
 */
const handleRevoke = async () => {
  try {
    await ElMessageBox.confirm(`确定要撤销证书 ${cert.value.domain} 吗？此操作不可恢复！`, '确认撤销', { type: 'error', confirmButtonText: '确认撤销' })
    await revokeCertificate(cert.value.id)
    ElMessage.success('证书已撤销')
    // 撤销后跳转回证书列表页
    router.push('/certificates')
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '撤销失败')
  }
}

// 组件挂载后加载证书详情
onMounted(loadCert)
</script>

<!-- 证书详情页面样式 -->
<style scoped>
/* 到期倒计时区域：居中显示 */
.countdown { text-align: center; padding: 20px 0; }
/* 大号倒计时数字：56px 加粗字体 */
.countdown-number { font-size: 56px; font-weight: 700; line-height: 1; }
/* 倒计时单位标签（"天"）：灰色小字 */
.countdown-label { font-size: 16px; color: #909399; margin-top: 8px; }
</style>
