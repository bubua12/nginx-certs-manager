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
            <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="证书" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.certificate" type="success" size="small">{{ row.certificate.domain }}</el-tag>
            <el-tag v-else-if="row.ssl_enabled" type="warning" size="small">未关联</el-tag>
            <span v-else style="color: #c0c4cc">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="$router.push(`/sites/${row.id}`)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[5, 10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadSites"
          @current-change="loadSites"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getSites } from '@/api/modules'

const sites = ref<any[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const loadSites = async () => {
  loading.value = true
  try {
    const res = await getSites(page.value, pageSize.value)
    sites.value = res.data.items || []
    total.value = res.data.total || 0
  } catch {
    ElMessage.error('加载站点列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadSites)
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
