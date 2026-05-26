<template>
  <div>
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>SSL 证书列表</span>
          <div class="actions">
            <el-button type="primary" @click="showRequestDialog = true">
              <el-icon><Plus /></el-icon>申请新证书
            </el-button>
            <el-button @click="loadCerts" :loading="loading">
              <el-icon><Refresh /></el-icon>刷新
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="certs" v-loading="loading" stripe>
        <el-table-column prop="domain" label="域名" min-width="200">
          <template #default="{ row }">
            <el-link type="primary" @click="$router.push(`/certificates/${row.id}`)">{{ row.domain }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="default">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="剩余天数" width="120" sortable :sort-method="(a: any, b: any) => a.days_left - b.days_left">
          <template #default="{ row }">
            <span :style="{ color: row.days_left <= 30 ? '#f56c6c' : '#303133', fontWeight: row.days_left <= 30 ? 'bold' : 'normal' }">
              {{ row.days_left }} 天
            </span>
          </template>
        </el-table-column>
        <el-table-column label="到期时间" width="140">
          <template #default="{ row }">{{ formatDate(row.not_after) }}</template>
        </el-table-column>
        <el-table-column prop="issuer" label="签发机构" width="180" show-overflow-tooltip />
        <el-table-column label="自动续期" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.auto_renew ? 'success' : 'info'" size="small">
              {{ row.auto_renew ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleRenew(row)" :loading="row._renewing">
              续期
            </el-button>
            <el-button type="danger" size="small" @click="handleRevoke(row)">
              撤销
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="showRequestDialog" title="申请新证书" width="500">
      <el-form :model="requestForm" label-width="80px">
        <el-form-item label="域名" required>
          <el-input v-model="requestForm.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="Webroot">
          <el-input v-model="requestForm.webroot" placeholder="/var/www/html (留空使用 standalone 模式)" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRequestDialog = false">取消</el-button>
        <el-button type="primary" @click="handleRequest" :loading="requesting">申请</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getCertificates, renewCertificate, requestCertificate, revokeCertificate } from '@/api/modules'
import dayjs from 'dayjs'

const certs = ref<any[]>([])
const loading = ref(false)
const showRequestDialog = ref(false)
const requesting = ref(false)
const requestForm = ref({ domain: '', webroot: '' })

const formatDate = (d: string) => dayjs(d).format('YYYY-MM-DD')

const statusType = (s: string) => ({ active: 'success', expiring: 'warning', expired: 'danger' }[s] || 'info') as any
const statusLabel = (s: string) => ({ active: '正常', expiring: '即将过期', expired: '已过期' }[s] || s)

const loadCerts = async () => {
  loading.value = true
  try {
    certs.value = (await getCertificates()).data || []
  } catch {
    ElMessage.error('加载证书列表失败')
  } finally {
    loading.value = false
  }
}

const handleRenew = async (cert: any) => {
  try {
    await ElMessageBox.confirm(`确定要续期证书 ${cert.domain} 吗？`, '确认续期', { type: 'warning' })
    cert._renewing = true
    const res = await renewCertificate(cert.id)
    ElMessage.success('续期成功')
    loadCerts()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '续期失败')
  } finally {
    cert._renewing = false
  }
}

const handleRevoke = async (cert: any) => {
  try {
    await ElMessageBox.confirm(`确定要撤销证书 ${cert.domain} 吗？此操作不可恢复！`, '确认撤销', { type: 'error', confirmButtonText: '确认撤销', cancelButtonText: '取消' })
    await revokeCertificate(cert.id)
    ElMessage.success('证书已撤销')
    loadCerts()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '撤销失败')
  }
}

const handleRequest = async () => {
  if (!requestForm.value.domain) {
    ElMessage.warning('请输入域名')
    return
  }
  requesting.value = true
  try {
    await requestCertificate(requestForm.value.domain, requestForm.value.webroot)
    ElMessage.success('证书申请已提交')
    showRequestDialog.value = false
    requestForm.value = { domain: '', webroot: '' }
    loadCerts()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '申请失败')
  } finally {
    requesting.value = false
  }
}

onMounted(loadCerts)
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.actions { display: flex; gap: 8px; }
</style>
