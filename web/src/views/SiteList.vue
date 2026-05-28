<!--
  站点列表页面模板 (SiteList.vue)
  功能：展示所有 Nginx 站点的列表，支持分页，可查看站点详情
-->
<template>
  <div>
    <!-- 站点列表主卡片 -->
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>Nginx 站点列表</span>
          <!-- 刷新按钮 -->
          <el-button @click="loadSites" :loading="loading">
            <el-icon><Refresh /></el-icon>刷新
          </el-button>
        </div>
      </template>

      <!-- 站点数据表格 -->
      <!-- v-loading: 加载时显示遮罩动画 -->
      <!-- stripe: 斑马纹样式 -->
      <el-table :data="sites" v-loading="loading" stripe>
        <!-- 域名列：可点击跳转到站点详情页 -->
        <el-table-column prop="domain" label="域名" min-width="200">
          <template #default="{ row }">
            <el-link type="primary" @click="$router.push(`/sites/${row.id}`)">{{ row.domain }}</el-link>
          </template>
        </el-table-column>
        <!-- 端口号列 -->
        <el-table-column prop="port" label="端口" width="80" />
        <!-- SSL 状态列：使用锁图标表示，绿色为已启用，灰色为未启用 -->
        <el-table-column label="SSL" width="80" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.ssl_enabled" color="#67c23a"><Lock /></el-icon>
            <el-icon v-else color="#c0c4cc"><Unlock /></el-icon>
          </template>
        </el-table-column>
        <!-- 反向代理地址列：显示 upstream 配置 -->
        <el-table-column prop="upstream" label="反向代理" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ row.upstream || '-' }}</template>
        </el-table-column>
        <!-- 启用状态列：绿色标签为启用，红色为禁用 -->
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <!-- 关联证书列：显示关联的 SSL 证书域名 -->
        <el-table-column label="证书" width="120">
          <template #default="{ row }">
            <!-- 已关联证书：显示证书域名 -->
            <el-tag v-if="row.certificate" type="success" size="small">{{ row.certificate.domain }}</el-tag>
            <!-- 开启了 SSL 但未关联证书：显示警告 -->
            <el-tag v-else-if="row.ssl_enabled" type="warning" size="small">未关联</el-tag>
            <!-- 未开启 SSL：显示占位符 -->
            <span v-else style="color: #c0c4cc">-</span>
          </template>
        </el-table-column>
        <!-- 操作列：查看详情按钮 -->
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="$router.push(`/sites/${row.id}`)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页组件 -->
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
/**
 * 站点列表页面逻辑 (SiteList.vue)
 * 功能：
 * 1. 加载并展示所有 Nginx 站点的分页列表
 * 2. 显示站点的域名、端口、SSL 状态、反向代理、启用状态和关联证书
 * 3. 提供分页切换功能
 */

// Vue 响应式 API
import { ref, onMounted } from 'vue'
// Element Plus 消息提示组件
import { ElMessage } from 'element-plus'
// 导入站点相关 API 接口
import { getSites } from '@/api/modules'

// ==================== 响应式状态 ====================

// 站点列表数据
const sites = ref<any[]>([])
// 表格加载状态
const loading = ref(false)
// 站点总数（用于分页组件）
const total = ref(0)
// 当前页码
const page = ref(1)
// 每页显示条数
const pageSize = ref(10)

// ==================== 数据加载函数 ====================

/**
 * 加载站点列表数据（分页）
 * 调用 getSites API，传入当前页码和每页条数
 */
const loadSites = async () => {
  loading.value = true
  try {
    const res = await getSites(page.value, pageSize.value)
    sites.value = res.data.items || []  // 更新站点列表
    total.value = res.data.total || 0   // 更新总数
  } catch {
    ElMessage.error('加载站点列表失败')
  } finally {
    loading.value = false
  }
}

// 组件挂载后加载站点列表
onMounted(loadSites)
</script>

<!-- 站点列表页面样式 -->
<style scoped>
/* 卡片头部：标题和刷新按钮分列两端 */
.card-header { display: flex; justify-content: space-between; align-items: center; }
/* 分页组件容器：右对齐，上方留间距 */
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
