<!-- 站点详情页面模板 -->
<!-- 功能：显示单个 Nginx 站点的详细信息，支持查看/编辑配置文件、启停站点、重载 Nginx -->
<template>
  <div v-loading="loading">
    <!-- 页面头部：返回按钮和站点域名标题 -->
    <el-page-header @back="$router.push('/sites')" :content="site?.domain || '站点详情'" style="margin-bottom: 20px" />

    <el-row :gutter="20">
      <!-- 左侧：站点基本信息卡片 -->
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header><span>站点信息</span></template>
          <!-- 站点属性描述列表：域名、端口、SSL 状态、运行状态、反向代理、配置路径 -->
          <el-descriptions :column="1" border>
            <el-descriptions-item label="域名">{{ site?.domain }}</el-descriptions-item>
            <el-descriptions-item label="端口">{{ site?.port }}</el-descriptions-item>
            <el-descriptions-item label="SSL">
              <!-- SSL 状态标签：绿色=已启用，灰色=未启用 -->
              <el-tag :type="site?.ssl_enabled ? 'success' : 'info'" size="small">
                {{ site?.ssl_enabled ? '已启用' : '未启用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <!-- 站点状态标签：绿色=已启用，红色=已禁用 -->
              <el-tag :type="site?.enabled ? 'success' : 'danger'" size="small">
                {{ site?.enabled ? '已启用' : '已禁用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="反向代理">{{ site?.upstream || '-' }}</el-descriptions-item>
            <el-descriptions-item label="配置路径">{{ site?.config_path || '-' }}</el-descriptions-item>
          </el-descriptions>

          <!-- 操作按钮区域：启用/禁用站点、重载 Nginx -->
          <div style="margin-top: 16px; display: flex; gap: 8px">
            <!-- 当前已启用则显示禁用按钮（warning 样式），反之显示启用按钮（success 样式） -->
            <el-button
              :type="site?.enabled ? 'warning' : 'success'"
              @click="handleToggle"
              :loading="toggling"
            >
              {{ site?.enabled ? '禁用站点' : '启用站点' }}
            </el-button>
            <el-button type="primary" @click="handleReload" :loading="reloading">
              重载 Nginx
            </el-button>
          </div>
        </el-card>
      </el-col>

      <!-- 右侧：Nginx 配置文件编辑卡片 -->
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>Nginx 配置文件</span>
              <div>
                <el-button type="primary" size="small" @click="handleSaveConfig" :loading="saving">
                  保存配置
                </el-button>
              </div>
            </div>
          </template>
          <!-- 配置文件编辑器：等宽字体，自动调整高度 -->
          <el-input
            v-model="configContent"
            type="textarea"
            :autosize="{ minRows: 20, maxRows: 40 }"
            placeholder="Nginx 配置内容"
            style="font-family: 'Courier New', monospace"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
/**
 * 站点详情页面逻辑
 * 功能：显示单个 Nginx 站点的详细信息，支持查看/编辑配置文件、启用/禁用站点、重载 Nginx
 * 路由参数：id - 站点 ID
 */

import { ref, onMounted } from 'vue' // 导入 Vue 3 组合式 API
import { useRoute } from 'vue-router' // 导入路由参数获取
import { ElMessage, ElMessageBox } from 'element-plus' // 导入 Element Plus 消息和确认框组件
import { getSite, getSiteConfig, updateSiteConfig, enableSite, disableSite, reloadNginx } from '@/api/modules' // 导入 API 接口

const route = useRoute() // 获取当前路由实例

// === 响应式状态 ===
const site = ref<any>(null)           // 站点详细信息对象
const configContent = ref('')          // Nginx 配置文件内容（可编辑文本）
const loading = ref(false)             // 页面加载状态
const toggling = ref(false)            // 启用/禁用操作加载状态
const reloading = ref(false)           // 重载 Nginx 操作加载状态
const saving = ref(false)              // 保存配置操作加载状态

/**
 * 加载站点数据和配置文件内容
 * 并行请求站点信息和配置文件，减少等待时间
 */
const loadData = async () => {
  loading.value = true
  try {
    const id = Number(route.params.id) // 从路由参数中获取站点 ID
    const [siteRes, configRes] = await Promise.all([getSite(id), getSiteConfig(id)]) // 并行请求
    site.value = siteRes.data            // 设置站点信息
    configContent.value = configRes.data.content // 设置配置文件内容
  } catch {
    ElMessage.error('加载站点信息失败')
  } finally {
    loading.value = false
  }
}

/**
 * 处理站点启用/禁用切换
 * 弹出确认框 -> 调用 API -> 重载 Nginx -> 刷新页面数据
 */
const handleToggle = async () => {
  const enable = !site.value.enabled // 反转当前状态
  const action = enable ? '启用' : '禁用'
  try {
    // 二次确认，防止误操作
    await ElMessageBox.confirm(`确定要${action}站点 ${site.value.domain} 吗？`, `确认${action}`, { type: 'warning' })
    toggling.value = true
    if (enable) await enableSite(site.value.id)   // 调用启用 API
    else await disableSite(site.value.id)          // 调用禁用 API
    await reloadNginx()                             // 重载 Nginx 使配置生效
    ElMessage.success(`站点已${action}`)
    loadData() // 刷新页面数据
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || `${action}失败`) // 忽略用户取消
  } finally {
    toggling.value = false
  }
}

/**
 * 处理 Nginx 重载操作
 * 不改变配置，仅重新加载 Nginx 进程配置
 */
const handleReload = async () => {
  reloading.value = true
  try {
    await reloadNginx()
    ElMessage.success('Nginx 已重新加载')
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '重载失败')
  } finally {
    reloading.value = false
  }
}

/**
 * 保存 Nginx 配置文件并重载 Nginx
 * 弹出确认提示（保存后需重载才生效） -> 保存配置 -> 重载 Nginx
 */
const handleSaveConfig = async () => {
  try {
    // 提示用户保存后需要重载
    await ElMessageBox.confirm('保存配置后需要重新加载 Nginx 才能生效，确定继续？', '确认保存', { type: 'warning' })
    saving.value = true
    await updateSiteConfig(site.value.id, configContent.value) // 保存配置文件内容
    await reloadNginx()                                         // 自动重载 Nginx
    ElMessage.success('配置已保存并重载 Nginx')
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '保存失败') // 忽略用户取消
  } finally {
    saving.value = false
  }
}

// 页面挂载后自动加载站点数据
onMounted(loadData)
</script>

<style scoped>
/* 配置文件卡片头部：标题和按钮水平排列，两端对齐 */
.card-header { display: flex; justify-content: space-between; align-items: center; }
</style>
