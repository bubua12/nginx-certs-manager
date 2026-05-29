<!--
  仪表盘页面模板 (Dashboard.vue)
  功能：展示系统整体概况，包括证书统计、站点统计、Nginx 状态、到期图表和操作日志
-->
<template>
  <div class="dashboard">
    <!-- ==================== 第一行：证书相关统计卡片 ==================== -->
    <el-row :gutter="20" class="stat-cards">
      <!-- 证书总数卡片 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <!-- 蓝色图标背景 -->
            <div class="stat-icon" style="background: #409eff22; color: #409eff">
              <el-icon :size="28"><Lock /></el-icon>
            </div>
            <div class="stat-info">
              <!-- 统计数值：显示证书总数 -->
              <div class="stat-value">{{ stats.total_certs }}</div>
              <div class="stat-label">证书总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <!-- 正常证书数量卡片 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <!-- 绿色图标背景 -->
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
      <!-- 即将过期证书数量卡片 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <!-- 橙色图标背景（警告色） -->
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
      <!-- 已过期证书数量卡片 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <!-- 红色图标背景（危险色） -->
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

    <!-- ==================== 第二行：站点和 Nginx 相关统计卡片 ==================== -->
    <el-row :gutter="20" style="margin-top: 20px">
      <!-- 站点总数卡片 -->
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
      <!-- 活跃站点数量卡片 -->
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
      <!-- Nginx 运行状态卡片 -->
      <!-- 图标颜色根据运行状态动态切换：运行中为绿色，已停止为红色 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" :style="{ background: nginxStatus?.running ? '#67c23a22' : '#f5623c22', color: nginxStatus?.running ? '#67c23a' : '#f56c6c' }">
              <el-icon :size="28"><Monitor /></el-icon>
            </div>
            <div class="stat-info">
              <!-- 文字颜色也根据运行状态动态切换 -->
              <div class="stat-value" :style="{ color: nginxStatus?.running ? '#67c23a' : '#f56c6c' }">
                {{ nginxStatus?.running ? '运行中' : '已停止' }}
              </div>
              <div class="stat-label">Nginx 状态</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <!-- SSL 站点数量卡片 -->
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <div class="stat-icon" style="background: #409eff22; color: #409eff">
              <el-icon :size="28"><Lock /></el-icon>
            </div>
            <div class="stat-info">
              <!-- ssl_sites 可能为 undefined，使用 || 0 提供默认值 -->
              <div class="stat-value">{{ stats.ssl_sites || 0 }}</div>
              <div class="stat-label">SSL 站点</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- ==================== 证书到期时间线图表 ==================== -->
    <!-- 使用 ECharts 柱状图展示各域名证书的剩余天数 -->
    <el-card shadow="hover" style="margin-top: 20px">
      <template #header>
        <span>证书到期时间线</span>
      </template>
      <!-- 有数据时渲染图表容器（高度 300px），无数据时显示空状态 -->
      <div v-if="timeline.length" ref="chartRef" style="height: 300px"></div>
      <el-empty v-else description="暂无证书数据" />
    </el-card>

    <!-- ==================== 最近操作日志表格 ==================== -->
    <el-card shadow="hover" style="margin-top: 20px">
      <template #header>
        <div class="card-header">
          <span>最近操作日志</span>
          <el-button type="primary" link @click="$router.push('/settings')">查看更多 →</el-button>
        </div>
      </template>
      <!-- 操作日志表格：stripe 带斑马纹，size="small" 紧凑模式 -->
      <el-table :data="logs" stripe size="small" max-height="300">
        <!-- 日志类型列：使用彩色标签显示 -->
        <el-table-column prop="type" label="类型" width="150">
          <template #default="{ row }">
            <el-tag size="small" :type="logTypeColor(row.type)">{{ logTypeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <!-- 操作目标列 -->
        <el-table-column prop="target" label="目标" width="200" />
        <!-- 状态列：成功为绿色，失败为红色 -->
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'success' ? 'success' : 'danger'">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <!-- 操作时间列 -->
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <!-- 详情列：超长文本以 tooltip 形式展示 -->
        <el-table-column prop="message" label="详情" show-overflow-tooltip />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
/**
 * 仪表盘页面逻辑 (Dashboard.vue)
 * 功能：
 * 1. 加载并展示证书统计数据（总数、正常、即将过期、已过期）
 * 2. 加载并展示站点统计数据和 Nginx 运行状态
 * 3. 使用 ECharts 渲染证书到期时间线柱状图
 * 4. 展示最近操作日志
 */

// Vue 响应式 API：ref 用于响应式变量，onMounted 在组件挂载后执行，nextTick 等待 DOM 更新
import { ref, onMounted, nextTick } from 'vue'
// 导入 ECharts 图表库（用于绘制证书到期时间线）
import * as echarts from 'echarts'
// 导入后端 API 接口函数
import { getDashboardStats, getDashboardTimeline, getNginxStatus, getLogs } from '@/api/modules'
// 导入 dayjs 日期格式化库
import dayjs from 'dayjs'

// ==================== 响应式状态 ====================

// 仪表盘统计数据对象，包含证书和站点的各种统计指标
// 初始值全部为 0，接口返回后会更新
const stats = ref({ total_certs: 0, active_certs: 0, expiring_soon: 0, expired_certs: 0, total_sites: 0, active_sites: 0, ssl_sites: 0 })
// 证书到期时间线数据（数组），用于 ECharts 图表渲染
const timeline = ref<any[]>([])
// Nginx 运行状态信息（运行中/已停止、版本号、PID 等）
const nginxStatus = ref<any>(null)
// 最近操作日志列表
const logs = ref<any[]>([])
// ECharts 图表容器 DOM 元素引用
const chartRef = ref<HTMLElement>()

// ==================== 工具函数 ====================

// 格式化时间为 "YYYY-MM-DD HH:mm" 格式
const formatTime = (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm')

/**
 * 将日志类型英文标识转换为中文显示名称
 * @param type - 日志类型标识（如 'cert_renew'）
 * @returns 中文名称（如 '证书续期'）
 */
const logTypeLabel = (type: string) => {
  const map: Record<string, string> = { cert_renew: '证书续期', cert_request: '申请证书', cert_revoke: '撤销证书', site_enable: '启用站点', site_disable: '禁用站点', site_config_update: '配置更新' }
  return map[type] || type
}

/**
 * 根据日志类型返回 Element Plus 标签的颜色类型
 * @param type - 日志类型标识
 * @returns Element Plus Tag 组件的 type 属性值（如 'warning'、'danger' 等）
 */
const logTypeColor = (type: string) => {
  const map: Record<string, string> = { cert_renew: 'warning', cert_request: 'primary', cert_revoke: 'danger', site_enable: 'success', site_disable: 'info' }
  return (map[type] || '') as any
}

// ==================== 数据加载函数 ====================

// 加载仪表盘统计数据（证书总数、各类证书数量、站点数量等）
const loadStats = async () => {
  try { stats.value = (await getDashboardStats()).data } catch {}
}

// 加载证书到期时间线数据，加载完成后渲染 ECharts 图表
const loadTimeline = async () => {
  try {
    timeline.value = (await getDashboardTimeline()).data || []
    // 等待 DOM 更新完成后再渲染图表（确保 chartRef 已挂载）
    await nextTick()
    renderChart()
  } catch {}
}

/**
 * 使用 ECharts 渲染证书到期时间线柱状图
 * 图表展示每个域名证书的剩余天数，颜色根据剩余天数变化：
 * - 红色：已过期（天数 < 0）
 * - 橙色：即将过期（天数 <= 30）
 * - 绿色：正常（天数 > 30）
 */
const renderChart = () => {
  // 如果图表容器不存在或没有数据，不渲染
  if (!chartRef.value || !timeline.value.length) return
  // 初始化 ECharts 实例
  const chart = echarts.init(chartRef.value)
  // 将时间线数据转换为图表所需格式，每个数据项带颜色信息
  const data = timeline.value.map((t: any) => ({
    value: t.days_left,        // 柱状图高度（剩余天数）
    name: t.domain,            // X 轴标签（域名）
    itemStyle: {
      // 根据剩余天数动态设置柱状图颜色
      color: t.days_left < 0 ? '#f56c6c' : t.days_left <= 30 ? '#e6a23c' : '#67c23a',
    },
  }))
  // 配置并渲染图表
  chart.setOption({
    // 提示框：鼠标悬停时显示域名和剩余天数
    tooltip: { trigger: 'axis', formatter: (p: any) => `${p[0].name}<br/>剩余 ${p[0].value} 天` },
    // X 轴：域名，标签旋转 30 度防止重叠
    xAxis: { type: 'category', data: data.map((d: any) => d.name), axisLabel: { rotate: 30 } },
    // Y 轴：剩余天数
    yAxis: { type: 'value', name: '剩余天数', axisLine: { show: true } },
    // 柱状图系列，最大宽度 40px
    series: [{ type: 'bar', data, barMaxWidth: 40 }],
    // 图表边距设置
    grid: { left: 60, right: 20, bottom: 60, top: 30 },
  })
}

// 加载 Nginx 运行状态信息
const loadNginxStatus = async () => {
  try { nginxStatus.value = (await getNginxStatus()).data } catch {}
}

// 加载最近操作日志（取前 10 条）
const loadLogs = async () => {
  try {
    const res = await getLogs(1, 5)
    logs.value = res.data.items || []
  } catch {}
}

// 组件挂载后，同时并行加载所有数据
onMounted(() => {
  loadStats()       // 加载统计数据
  loadTimeline()    // 加载时间线数据并渲染图表
  loadNginxStatus() // 加载 Nginx 状态
  loadLogs()        // 加载操作日志
})
</script>

<!-- 仪表盘页面样式 -->
<style scoped>
/* 统计卡片行：鼠标悬停时显示手型光标 */
.stat-cards .el-card { cursor: pointer; }
/* 统计卡片内部布局：图标和文字水平排列 */
.stat-card {
  display: flex;
  align-items: center;
  gap: 16px; /* 图标和文字之间的间距 */
}
/* 统计图标容器：固定大小的圆角方块，内含彩色图标 */
.stat-icon {
  width: 56px; height: 56px;
  border-radius: 12px; /* 圆角 */
  display: flex; align-items: center; justify-content: center;
}
/* 统计数值样式：大号加粗数字 */
.stat-value { font-size: 28px; font-weight: 600; color: #303133; }
/* 统计标签样式：小号灰色描述文字 */
.stat-label { font-size: 14px; color: #909399; margin-top: 4px; }
/* 卡片头部：标题和按钮分列两端 */
.card-header { display: flex; justify-content: space-between; align-items: center; }
</style>
