<template>
  <div v-loading="loading">
    <el-page-header @back="$router.push('/certificates')" :content="cert?.domain || '证书详情'" style="margin-bottom: 20px" />

    <el-row :gutter="20">
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header><span>证书信息</span></template>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="域名">{{ cert?.domain }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="statusType(cert?.status)">{{ statusLabel(cert?.status) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="签发机构">{{ cert?.issuer || '-' }}</el-descriptions-item>
            <el-descriptions-item label="自动续期">
              <el-tag :type="cert?.auto_renew ? 'success' : 'info'" size="small">
                {{ cert?.auto_renew ? '是' : '否' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="生效时间">{{ formatDate(cert?.not_before) }}</el-descriptions-item>
            <el-descriptions-item label="过期时间">{{ formatDate(cert?.not_after) }}</el-descriptions-item>
            <el-descriptions-item label="剩余天数">
              <span :style="{ color: (cert?.days_left || 0) <= 30 ? '#f56c6c' : '#67c23a', fontWeight: 'bold', fontSize: '16px' }">
                {{ cert?.days_left ?? '-' }} 天
              </span>
            </el-descriptions-item>
            <el-descriptions-item label="上次续期">{{ formatDate(cert?.last_renewed) || '-' }}</el-descriptions-item>
            <el-descriptions-item label="证书路径" :span="2">{{ cert?.cert_path || '-' }}</el-descriptions-item>
            <el-descriptions-item label="私钥路径" :span="2">{{ cert?.key_path || '-' }}</el-descriptions-item>
            <el-descriptions-item label="SAN 域名" :span="2">
              <el-tag v-for="san in parsedSANs" :key="san" style="margin: 2px 4px" size="small">{{ san }}</el-tag>
              <span v-if="!parsedSANs.length">-</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="hover">
          <template #header><span>操作</span></template>
          <div style="display: flex; flex-direction: column; gap: 12px">
            <el-button type="primary" size="large" @click="handleRenew" :loading="renewing">
              <el-icon><Refresh /></el-icon>立即续期
            </el-button>
            <el-button type="danger" size="large" @click="handleRevoke">
              <el-icon><Delete /></el-icon>撤销证书
            </el-button>
          </div>
        </el-card>

        <el-card shadow="hover" style="margin-top: 20px">
          <template #header><span>到期倒计时</span></template>
          <div class="countdown">
            <div class="countdown-number" :style="{ color: countdownColor }">
              {{ cert?.days_left ?? '-' }}
            </div>
            <div class="countdown-label">天</div>
          </div>
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
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getCertificate, renewCertificate, revokeCertificate } from '@/api/modules'
import dayjs from 'dayjs'

const route = useRoute()
const router = useRouter()
const cert = ref<any>(null)
const loading = ref(false)
const renewing = ref(false)

const formatDate = (d: string) => {
  if (!d) return '-'
  const date = new Date(d)
  return date.getFullYear() > 2000 ? dayjs(d).format('YYYY-MM-DD HH:mm') : '-'
}
const statusType = (s: string) => ({ active: 'success', expiring: 'warning', expired: 'danger' }[s] || 'info') as any
const statusLabel = (s: string) => ({ active: '正常', expiring: '即将过期', expired: '已过期' }[s] || s || '-')

const parsedSANs = computed(() => {
  try { return cert.value?.sans ? JSON.parse(cert.value.sans) : [] } catch { return [] }
})

const countdownColor = computed(() => {
  const d = cert.value?.days_left ?? 0
  return d < 0 ? '#f56c6c' : d <= 30 ? '#e6a23c' : '#67c23a'
})

const progressPercent = computed(() => {
  const d = cert.value?.days_left ?? 0
  return Math.max(0, Math.min(100, (d / 90) * 100))
})

const progressColor = computed(() => {
  const d = cert.value?.days_left ?? 0
  return d < 0 ? '#f56c6c' : d <= 30 ? '#e6a23c' : '#67c23a'
})

const loadCert = async () => {
  loading.value = true
  try {
    const res = await getCertificate(Number(route.params.id))
    const d = res.data
    const expiry = new Date(d.not_after)
    const daysLeft = expiry.getFullYear() > 2000
      ? Math.floor((expiry.getTime() - Date.now()) / 86400000)
      : -99999
    cert.value = { ...d, days_left: daysLeft }
  } catch {
    ElMessage.error('加载证书信息失败')
  } finally {
    loading.value = false
  }
}

const handleRenew = async () => {
  try {
    await ElMessageBox.confirm(`确定要续期证书 ${cert.value.domain} 吗？`, '确认续期', { type: 'warning' })
    renewing.value = true
    await renewCertificate(cert.value.id)
    ElMessage.success('续期成功')
    loadCert()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '续期失败')
  } finally {
    renewing.value = false
  }
}

const handleRevoke = async () => {
  try {
    await ElMessageBox.confirm(`确定要撤销证书 ${cert.value.domain} 吗？此操作不可恢复！`, '确认撤销', { type: 'error', confirmButtonText: '确认撤销' })
    await revokeCertificate(cert.value.id)
    ElMessage.success('证书已撤销')
    router.push('/certificates')
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '撤销失败')
  }
}

onMounted(loadCert)
</script>

<style scoped>
.countdown { text-align: center; padding: 20px 0; }
.countdown-number { font-size: 56px; font-weight: 700; line-height: 1; }
.countdown-label { font-size: 16px; color: #909399; margin-top: 8px; }
</style>
