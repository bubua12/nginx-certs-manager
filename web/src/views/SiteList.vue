<template>
  <div>
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>Nginx 站点列表</span>
          <el-button @click="loadSites" :loading="loading">
            <el-icon><Refresh /></el-icon>刷新
          </el-button>
        </div>
      </template>

      <el-table :data="sites" v-loading="loading" stripe>
        <el-table-column prop="domain" label="域名" min-width="200">
          <template #default="{ row }">
            <el-link type="primary" @click="$router.push(`/sites/${row.id}`)">{{ row.domain }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="port" label="端口" width="80" />
        <el-table-column label="SSL" width="80" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.ssl_enabled" color="#67c23a"><Lock /></el-icon>
            <el-icon v-else color="#c0c4cc"><Unlock /></el-icon>
          </template>
        </el-table-column>
        <el-table-column prop="upstream" label="反向代理" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ row.upstream || '-' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              @change="(val: boolean) => handleToggle(row, val)"
              :loading="row._toggling"
              active-text="启用"
              inactive-text="禁用"
              inline-prompt
            />
          </template>
        </el-table-column>
        <el-table-column label="证书" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.certificate" type="success" size="small">{{ row.certificate.domain }}</el-tag>
            <el-tag v-else-if="row.ssl_enabled" type="warning" size="small">未关联</el-tag>
            <span v-else style="color: #c0c4cc">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="$router.push(`/sites/${row.id}`)">
              详情
            </el-button>
            <el-button size="small" @click="handleValidate" :loading="validating">
              校验
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getSites, enableSite, disableSite, validateNginx, reloadNginx } from '@/api/modules'

const sites = ref<any[]>([])
const loading = ref(false)
const validating = ref(false)

const loadSites = async () => {
  loading.value = true
  try {
    sites.value = (await getSites()).data || []
  } catch {
    ElMessage.error('加载站点列表失败')
  } finally {
    loading.value = false
  }
}

const handleToggle = async (site: any, enable: boolean) => {
  const action = enable ? '启用' : '禁用'
  try {
    await ElMessageBox.confirm(`确定要${action}站点 ${site.domain} 吗？`, `确认${action}`, { type: 'warning' })
    site._toggling = true
    if (enable) {
      await enableSite(site.id)
    } else {
      await disableSite(site.id)
    }
    // Auto reload nginx
    await reloadNginx()
    ElMessage.success(`站点已${action}，Nginx 已重新加载`)
  } catch (e: any) {
    site.enabled = !enable
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || `${action}失败`)
  } finally {
    site._toggling = false
  }
}

const handleValidate = async () => {
  validating.value = true
  try {
    const res = await validateNginx()
    if (res.data.valid) {
      ElMessage.success('Nginx 配置校验通过')
    } else {
      ElMessage.warning(`配置校验结果:\n${res.data.output}`)
    }
  } catch (e: any) {
    ElMessage.error('校验失败')
  } finally {
    validating.value = false
  }
}

onMounted(loadSites)
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
</style>
