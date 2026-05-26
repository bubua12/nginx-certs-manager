<template>
  <div v-loading="loading">
    <el-page-header @back="$router.push('/sites')" :content="site?.domain || '站点详情'" style="margin-bottom: 20px" />

    <el-row :gutter="20">
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header><span>站点信息</span></template>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="域名">{{ site?.domain }}</el-descriptions-item>
            <el-descriptions-item label="端口">{{ site?.port }}</el-descriptions-item>
            <el-descriptions-item label="SSL">
              <el-tag :type="site?.ssl_enabled ? 'success' : 'info'" size="small">
                {{ site?.ssl_enabled ? '已启用' : '未启用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="site?.enabled ? 'success' : 'danger'" size="small">
                {{ site?.enabled ? '已启用' : '已禁用' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="反向代理">{{ site?.upstream || '-' }}</el-descriptions-item>
            <el-descriptions-item label="配置路径">{{ site?.config_path || '-' }}</el-descriptions-item>
          </el-descriptions>

          <div style="margin-top: 16px; display: flex; gap: 8px">
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
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getSite, getSiteConfig, updateSiteConfig, enableSite, disableSite, reloadNginx } from '@/api/modules'

const route = useRoute()
const site = ref<any>(null)
const configContent = ref('')
const loading = ref(false)
const toggling = ref(false)
const reloading = ref(false)
const saving = ref(false)

const loadData = async () => {
  loading.value = true
  try {
    const id = Number(route.params.id)
    const [siteRes, configRes] = await Promise.all([getSite(id), getSiteConfig(id)])
    site.value = siteRes.data
    configContent.value = configRes.data.content
  } catch {
    ElMessage.error('加载站点信息失败')
  } finally {
    loading.value = false
  }
}

const handleToggle = async () => {
  const enable = !site.value.enabled
  const action = enable ? '启用' : '禁用'
  try {
    await ElMessageBox.confirm(`确定要${action}站点 ${site.value.domain} 吗？`, `确认${action}`, { type: 'warning' })
    toggling.value = true
    if (enable) await enableSite(site.value.id)
    else await disableSite(site.value.id)
    await reloadNginx()
    ElMessage.success(`站点已${action}`)
    loadData()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || `${action}失败`)
  } finally {
    toggling.value = false
  }
}

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

const handleSaveConfig = async () => {
  try {
    await ElMessageBox.confirm('保存配置后需要重新加载 Nginx 才能生效，确定继续？', '确认保存', { type: 'warning' })
    saving.value = true
    await updateSiteConfig(site.value.id, configContent.value)
    await reloadNginx()
    ElMessage.success('配置已保存并重载 Nginx')
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
</style>
