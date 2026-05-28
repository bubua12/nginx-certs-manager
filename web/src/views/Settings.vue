<!-- 系统设置页面模板 -->
<!-- 功能：展示 Nginx 系统信息、配置系统参数（扫描间隔/通知方式/过期预警）、查看操作日志 -->
<template>
  <div>
    <el-row :gutter="20">
      <!-- 左侧：Nginx 系统信息卡片 -->
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header><span>系统信息</span></template>
          <!-- Nginx 运行状态描述列表 -->
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Nginx 状态">
              <!-- 状态标签：绿色=运行中，红色=已停止 -->
              <el-tag :type="nginxStatus?.running ? 'success' : 'danger'">
                {{ nginxStatus?.running ? '运行中' : '已停止' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Nginx 版本">{{ nginxStatus?.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Nginx PID">{{ nginxStatus?.pid || '-' }}</el-descriptions-item>
          </el-descriptions>

          <!-- Nginx 操作按钮区域 -->
          <div style="margin-top: 16px; display: flex; gap: 8px">
            <el-button type="primary" @click="handleReload" :loading="reloading">重载 Nginx</el-button>
            <el-button @click="handleValidate" :loading="validating">校验配置</el-button>
          </div>
        </el-card>
      </el-col>

      <!-- 右侧：系统设置表单卡片 -->
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header><span>设置</span></template>
          <el-form label-width="120px">
            <!-- 自动扫描间隔设置 -->
            <el-form-item label="自动扫描间隔">
              <el-select v-model="settings.scan_interval" style="width: 100%">
                <el-option label="15 分钟" value="15m" />
                <el-option label="30 分钟" value="30m" />
                <el-option label="1 小时" value="1h" />
                <el-option label="6 小时" value="6h" />
                <el-option label="24 小时" value="24h" />
              </el-select>
            </el-form-item>
            <!-- 通知方式设置 -->
            <el-form-item label="通知方式">
              <el-select v-model="settings.notify_type" style="width: 100%">
                <el-option label="关闭" value="none" />
                <el-option label="邮件" value="email" />
                <el-option label="Webhook" value="webhook" />
              </el-select>
            </el-form-item>
            <!-- Webhook URL 输入框（仅在通知方式为 webhook 时显示） -->
            <el-form-item v-if="settings.notify_type === 'webhook'" label="Webhook URL">
              <el-input v-model="settings.webhook_url" placeholder="https://hooks.example.com/..." />
            </el-form-item>
            <!-- 证书过期预警天数设置 -->
            <el-form-item label="过期预警天数">
              <el-input-number v-model.number="settings.warn_days" :min="1" :max="90" />
            </el-form-item>
            <!-- 保存设置按钮 -->
            <el-form-item>
              <el-button type="primary" @click="handleSaveSettings" :loading="saving">保存设置</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- 操作日志表格卡片 -->
    <el-card shadow="hover" style="margin-top: 20px">
      <template #header><span>操作日志</span></template>
      <!-- 日志列表：类型标签、目标、状态标签、时间、详情 -->
      <el-table :data="logs" stripe size="small" max-height="400">
        <el-table-column prop="type" label="类型" width="150">
          <template #default="{ row }">
            <!-- 日志类型标签，不同操作类型显示不同颜色 -->
            <el-tag size="small" :type="logTypeColor(row.type)">{{ logTypeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target" label="目标" width="200" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <!-- 操作状态标签：绿色=成功，红色=失败 -->
            <el-tag size="small" :type="row.status === 'success' ? 'success' : 'danger'">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="message" label="详情" show-overflow-tooltip /> <!-- 溢出文本以 tooltip 显示 -->
      </el-table>

      <!-- 分页组件 -->
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
/**
 * 系统设置页面逻辑
 * 功能：显示 Nginx 系统信息、管理扫描间隔和通知设置、查看操作日志
 * 包含：Nginx 重载/校验、设置保存、分页日志查询
 */

import { ref, reactive, onMounted } from 'vue' // 导入 Vue 3 组合式 API
import { ElMessage } from 'element-plus' // 导入 Element Plus 消息提示组件
import { getNginxStatus, reloadNginx, validateNginx, getSettings, updateSettings, getLogs } from '@/api/modules' // 导入 API 接口
import dayjs from 'dayjs' // 导入日期格式化库

// === 响应式状态 ===
const nginxStatus = ref<any>(null)   // Nginx 运行状态信息（运行中/版本/PID）
const reloading = ref(false)          // 重载 Nginx 操作加载状态
const validating = ref(false)         // 校验配置操作加载状态
const saving = ref(false)             // 保存设置操作加载状态
const logs = ref<any[]>([])           // 操作日志列表数据
const logTotal = ref(0)               // 日志总条数（用于分页）
const logPage = ref(1)                // 当前日志页码
const logPageSize = ref(10)           // 每页显示日志条数

// 系统设置响应式对象
const settings = reactive({
  scan_interval: '30m',      // 自动扫描间隔（默认 30 分钟）
  notify_type: 'none',       // 通知方式：none/email/webhook
  webhook_url: '',           // Webhook 回调地址
  warn_days: 30,             // 证书过期预警天数（提前多少天告警）
})

// 格式化时间：将 ISO 时间字符串转为 "YYYY-MM-DD HH:mm" 格式
const formatTime = (t: string) => dayjs(t).format('YYYY-MM-DD HH:mm')

/**
 * 日志类型中文标签映射
 * 将后端返回的操作类型标识符转为用户友好的中文名称
 */
const logTypeLabel = (type: string) => {
  const map: Record<string, string> = { cert_renew: '证书续期', cert_request: '申请证书', cert_revoke: '撤销证书', site_enable: '启用站点', site_disable: '禁用站点', site_config_update: '配置更新' }
  return map[type] || type // 未知类型直接返回原始标识符
}

/**
 * 日志类型颜色映射
 * 不同操作类型使用不同颜色的标签，便于视觉区分
 */
const logTypeColor = (type: string) => {
  const map: Record<string, string> = { cert_renew: 'warning', cert_request: 'primary', cert_revoke: 'danger', site_enable: 'success', site_disable: 'info' }
  return (map[type] || '') as any // 未知类型使用默认颜色
}

/**
 * 加载 Nginx 运行状态信息
 * 获取版本号、PID、运行状态等
 */
const loadNginxStatus = async () => {
  try { nginxStatus.value = (await getNginxStatus()).data } catch {} // 请求失败静默处理
}

/**
 * 处理 Nginx 重载操作
 * 向 Nginx 发送 reload 信号，使其重新加载配置文件
 */
const handleReload = async () => {
  reloading.value = true
  try {
    await reloadNginx()
    ElMessage.success('Nginx 已重新加载')
    loadNginxStatus() // 重载后刷新状态信息
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '重载失败')
  } finally {
    reloading.value = false
  }
}

/**
 * 处理 Nginx 配置校验操作
 * 执行 nginx -t 命令检查配置文件语法是否正确
 */
const handleValidate = async () => {
  validating.value = true
  try {
    const res = await validateNginx()
    if (res.data.valid) ElMessage.success('Nginx 配置校验通过')    // 校验通过
    else ElMessage.warning(res.data.output)                        // 校验失败，显示错误详情
  } catch {
    ElMessage.error('校验失败')
  } finally {
    validating.value = false
  }
}

/**
 * 从后端加载系统设置
 * 将返回的设置数据合并到本地 reactive 对象中
 */
const loadSettings = async () => {
  try {
    const res = await getSettings()
    Object.assign(settings, res.data) // 用后端数据覆盖本地默认值
  } catch {} // 请求失败保持默认值
}

/**
 * 保存系统设置
 * 将所有设置值转为字符串后发送到后端（后端统一存储为字符串类型）
 */
const handleSaveSettings = async () => {
  saving.value = true
  try {
    const payload: Record<string, string> = {}
    for (const [key, val] of Object.entries(settings)) {
      payload[key] = String(val) // 统一转为字符串，后端 key-value 存储要求
    }
    await updateSettings(payload)
    ElMessage.success('设置已保存')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

/**
 * 加载操作日志（支持分页）
 * 根据当前页码和每页条数请求日志数据
 */
const loadLogs = async () => {
  try {
    const res = await getLogs(logPage.value, logPageSize.value)
    logs.value = res.data.items || []     // 日志列表数据
    logTotal.value = res.data.total || 0  // 更新总条数
  } catch {} // 请求失败保持空列表
}

// 页面挂载后并行加载所有数据
onMounted(() => {
  loadNginxStatus() // 加载 Nginx 状态
  loadSettings()    // 加载系统设置
  loadLogs()        // 加载操作日志
})
</script>

<style scoped>
/* 分页组件容器：右对齐，与日志表格保持间距 */
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
