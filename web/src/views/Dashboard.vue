<template>
  <div class="dashboard">
    <el-row :gutter="20" class="stat-cards">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #409eff22; color: #409eff">
              <el-icon :size="28"><Lock /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total_certs }}</div>
              <div class="stat-label">证书总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #67c23a22; color: #67c23a">
              <el-icon :size="28"><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.active_certs }}</div>
              <div class="stat-label">正常证书</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #e6a23c22; color: #e6a23c">
              <el-icon :size="28"><Warning /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.expiring_soon }}</div>
              <div class="stat-label">即将过期</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #f5623c22; color: #f56c6c">
              <el-icon :size="28"><CircleClose /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.expired_certs }}</div>
              <div class="stat-label">已过期</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #90939922; color: #909399">
              <el-icon :size="28"><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total_sites }}</div>
              <div class="stat-label">站点总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #67c23a22; color: #67c23a">
              <el-icon :size="28"><VideoPlay /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.active_sites }}</div>
              <div class="stat-label">活跃站点</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>Nginx 状态</span>
              <el-button type="primary" size="small" @click="reloadNginxHandler" :loading="nginxReloading">
                重新加载
              </el-button>
            </div>
          </template>
          <div v-if="nginxStatus" class="nginx-status">
            <el-tag :type="nginxStatus.running ? 'success' : 'danger'" size="large">
              {{ nginxStatus.running ? '运行中' : '已停止' }}
            </el-tag>
            <span v-if="nginxStatus.version" style="margin-left: 16px; color: #666">
              v{{ nginxStatus.version }}
            </span>
            <span v-if="nginxStatus.pid" style="margin-left: 16px; color: #999">
              PID: {{ nginxStatus.pid }}
            </span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover" style="margin-top: 20px">
      <template #header>
        <span>证书到期时间线</span>
      </template>
      <div v-if="timeline.length" ref="chartRef" style="height: 300px"></div>
      <el-empty v-else description="暂无证书数据" />
    </el-card>

    <el-card shadow="hover" style="margin-top: 20px">
      <template #header>
        <span>最近操作日志</span>
      </template>
      <el-table :data="logs" stripe size="small" max-height="300">
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
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'
import { getDashboardStats, getDashboardTimeline, getNginxStatus, reloadNginx, getLogs } from '@/api/modules'
import dayjs from 'dayjs'

const stats = ref({ total_certs: 0, active_certs: 0, expiring_soon: 0, expired_certs: 0, total_sites: 0, active_sites: 0 })
const timeline = ref<any[]>([])
const nginxStatus = ref<any>(null)
const nginxReloading = ref(false)
const logs = ref<any[]>([])
const chartRef = ref<HTMLElement>()

const formatTime = (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm')

const logTypeLabel = (type: string) => {
  const map: Record<string, string> = { cert_renew: '证书续期', cert_request: '申请证书', cert_revoke: '撤销证书', site_enable: '启用站点', site_disable: '禁用站点', site_config_update: '配置更新' }
  return map[type] || type
}
const logTypeColor = (type: string) => {
  const map: Record<string, string> = { cert_renew: 'warning', cert_request: 'primary', cert_revoke: 'danger', site_enable: 'success', site_disable: 'info' }
  return (map[type] || '') as any
}

const loadStats = async () => {
  try { stats.value = (await getDashboardStats()).data } catch {}
}

const loadTimeline = async () => {
  try {
    timeline.value = (await getDashboardTimeline()).data || []
    await nextTick()
    renderChart()
  } catch {}
}

const renderChart = () => {
  if (!chartRef.value || !timeline.value.length) return
  const chart = echarts.init(chartRef.value)
  const data = timeline.value.map((t: any) => ({
    value: t.days_left,
    name: t.domain,
    itemStyle: {
      color: t.days_left < 0 ? '#f56c6c' : t.days_left <= 30 ? '#e6a23c' : '#67c23a',
    },
  }))
  chart.setOption({
    tooltip: { trigger: 'axis', formatter: (p: any) => `${p[0].name}<br/>剩余 ${p[0].value} 天` },
    xAxis: { type: 'category', data: data.map((d: any) => d.name), axisLabel: { rotate: 30 } },
    yAxis: { type: 'value', name: '剩余天数', axisLine: { show: true } },
    series: [{ type: 'bar', data, barMaxWidth: 40 }],
    grid: { left: 60, right: 20, bottom: 60, top: 30 },
  })
}

const loadNginxStatus = async () => {
  try { nginxStatus.value = (await getNginxStatus()).data } catch {}
}

const reloadNginxHandler = async () => {
  nginxReloading.value = true
  try {
    await reloadNginx()
    ElMessage.success('Nginx 已重新加载')
    await loadNginxStatus()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '加载失败')
  } finally {
    nginxReloading.value = false
  }
}

const loadLogs = async () => {
  try {
    const res = await getLogs(1, 10)
    logs.value = res.data.items || []
  } catch {}
}

onMounted(() => {
  loadStats()
  loadTimeline()
  loadNginxStatus()
  loadLogs()
})
</script>

<style scoped>
.stat-cards .el-card { cursor: pointer; }
.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
}
.stat-icon {
  width: 56px; height: 56px;
  border-radius: 12px;
  display: flex; align-items: center; justify-content: center;
}
.stat-value { font-size: 28px; font-weight: 600; color: #303133; }
.stat-label { font-size: 14px; color: #909399; margin-top: 4px; }
.card-header {
  display: flex; justify-content: space-between; align-items: center;
}
.nginx-status {
  display: flex; align-items: center;
}
</style>
