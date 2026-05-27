<template>
  <div>
    <el-row :gutter="20">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header><span>系统信息</span></template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Nginx 状态">
              <el-tag :type="nginxStatus?.running ? 'success' : 'danger'">
                {{ nginxStatus?.running ? '运行中' : '已停止' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Nginx 版本">{{ nginxStatus?.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Nginx PID">{{ nginxStatus?.pid || '-' }}</el-descriptions-item>
          </el-descriptions>

          <div style="margin-top: 16px; display: flex; gap: 8px">
            <el-button type="primary" @click="handleReload" :loading="reloading">重载 Nginx</el-button>
            <el-button @click="handleValidate" :loading="validating">校验配置</el-button>
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="hover">
          <template #header><span>设置</span></template>
          <el-form label-width="120px">
            <el-form-item label="自动扫描间隔">
              <el-select v-model="settings.scan_interval" style="width: 100%">
                <el-option label="15 分钟" value="15m" />
                <el-option label="30 分钟" value="30m" />
                <el-option label="1 小时" value="1h" />
                <el-option label="6 小时" value="6h" />
                <el-option label="24 小时" value="24h" />
              </el-select>
            </el-form-item>
            <el-form-item label="通知方式">
              <el-select v-model="settings.notify_type" style="width: 100%">
                <el-option label="关闭" value="none" />
                <el-option label="邮件" value="email" />
                <el-option label="Webhook" value="webhook" />
              </el-select>
            </el-form-item>
            <el-form-item v-if="settings.notify_type === 'webhook'" label="Webhook URL">
              <el-input v-model="settings.webhook_url" placeholder="https://hooks.example.com/..." />
            </el-form-item>
            <el-form-item label="过期预警天数">
              <el-input-number v-model.number="settings.warn_days" :min="1" :max="90" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSaveSettings" :loading="saving">保存设置</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover" style="margin-top: 20px">
      <template #header><span>操作日志</span></template>
      <el-table :data="logs" stripe size="small" max-height="400">
        <el-table-column prop="type" label="类型" width="150">
          <template #default="{ row }">
            <el-tag size="small" :type="logTypeColor(row.type)">{{ logTypeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target" label="目标" width="200" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'success' ? 'success' : 'danger'">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="message" label="详情" show-overflow-tooltip />
      </el-table>

      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="logPage"
          v-model:page-size="logPageSize"
          :page-sizes="[5, 10, 20, 50]"
          :total="logTotal"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadLogs"
          @current-change="loadLogs"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getNginxStatus, reloadNginx, validateNginx, getSettings, updateSettings, getLogs } from '@/api/modules'
import dayjs from 'dayjs'

const nginxStatus = ref<any>(null)
const reloading = ref(false)
const validating = ref(false)
const saving = ref(false)
const logs = ref<any[]>([])
const logTotal = ref(0)
const logPage = ref(1)
const logPageSize = ref(10)

const settings = reactive({
  scan_interval: '30m',
  notify_type: 'none',
  webhook_url: '',
  warn_days: 30,
})

const formatTime = (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm')
const logTypeLabel = (type: string) => {
  const map: Record<string, string> = { cert_renew: '证书续期', cert_request: '申请证书', cert_revoke: '撤销证书', site_enable: '启用站点', site_disable: '禁用站点', site_config_update: '配置更新' }
  return map[type] || type
}
const logTypeColor = (type: string) => {
  const map: Record<string, string> = { cert_renew: 'warning', cert_request: 'primary', cert_revoke: 'danger', site_enable: 'success', site_disable: 'info' }
  return (map[type] || '') as any
}

const loadNginxStatus = async () => {
  try { nginxStatus.value = (await getNginxStatus()).data } catch {}
}

const handleReload = async () => {
  reloading.value = true
  try {
    await reloadNginx()
    ElMessage.success('Nginx 已重新加载')
    loadNginxStatus()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '重载失败')
  } finally {
    reloading.value = false
  }
}

const handleValidate = async () => {
  validating.value = true
  try {
    const res = await validateNginx()
    if (res.data.valid) ElMessage.success('Nginx 配置校验通过')
    else ElMessage.warning(res.data.output)
  } catch {
    ElMessage.error('校验失败')
  } finally {
    validating.value = false
  }
}

const loadSettings = async () => {
  try {
    const res = await getSettings()
    Object.assign(settings, res.data)
  } catch {}
}

const handleSaveSettings = async () => {
  saving.value = true
  try {
    const payload: Record<string, string> = {}
    for (const [key, val] of Object.entries(settings)) {
      payload[key] = String(val)
    }
    await updateSettings(payload)
    ElMessage.success('设置已保存')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const loadLogs = async () => {
  try {
    const res = await getLogs(logPage.value, logPageSize.value)
    logs.value = res.data.items || []
    logTotal.value = res.data.total || 0
  } catch {}
}

onMounted(() => {
  loadNginxStatus()
  loadSettings()
  loadLogs()
})
</script>

<style scoped>
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
