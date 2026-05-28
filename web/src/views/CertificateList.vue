<!--
  证书列表页面模板 (CertificateList.vue)
  功能：展示所有 SSL 证书的列表，支持分页、续期、撤销和申请新证书
-->
<template>
  <div>
    <!-- 证书列表主卡片 -->
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>SSL 证书列表</span>
          <div class="actions">
            <!-- 申请新证书按钮：点击打开申请对话框 -->
            <el-button type="primary" @click="showRequestDialog = true">
              <el-icon><Plus /></el-icon>申请新证书
            </el-button>
            <!-- 刷新按钮：重新加载证书列表 -->
            <el-button @click="loadCerts" :loading="loading">
              <el-icon><Refresh /></el-icon>刷新
            </el-button>
          </div>
        </div>
      </template>

      <!-- 证书数据表格 -->
      <!-- :data: 绑定证书列表数据 -->
      <!-- v-loading: 加载时显示遮罩动画 -->
      <!-- stripe: 斑马纹样式，隔行变色 -->
      <el-table :data="certs" v-loading="loading" stripe>
        <!-- 域名列：可点击跳转到证书详情页 -->
        <el-table-column prop="domain" label="域名" min-width="200">
          <template #default="{ row }">
            <el-link type="primary" @click="$router.push(`/certificates/${row.id}`)">{{ row.domain }}</el-link>
          </template>
        </el-table-column>
        <!-- 状态列：使用彩色标签显示（正常/即将过期/已过期） -->
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <!-- 剩余天数列：支持排序，30 天内标红加粗 -->
        <el-table-column label="剩余天数" width="120" sortable :sort-method="(a: any, b: any) => a.days_left - b.days_left">
          <template #default="{ row }">
            <!-- 剩余天数 <= 30 时显示红色加粗，提醒用户注意 -->
            <span :style="{ color: row.days_left <= 30 ? '#f56c6c' : '#303133', fontWeight: row.days_left <= 30 ? 'bold' : 'normal' }">
              {{ row.days_left }} 天
            </span>
          </template>
        </el-table-column>
        <!-- 到期时间列 -->
        <el-table-column label="到期时间" width="140">
          <template #default="{ row }">{{ formatDate(row.not_after) }}</template>
        </el-table-column>
        <!-- 签发机构列：超长文本以 tooltip 形式展示 -->
        <el-table-column prop="issuer" label="签发机构" width="180" show-overflow-tooltip />
        <!-- 自动续期列：显示"是"（绿色标签）或"否"（灰色标签） -->
        <el-table-column label="自动续期" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.auto_renew ? 'success' : 'info'" size="small">{{ row.auto_renew ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <!-- 操作列：续期和撤销按钮 -->
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <!-- 续期按钮：:loading 通过 row._renewing 控制每行独立的加载状态 -->
            <el-button type="primary" size="small" @click="handleRenew(row)" :loading="row._renewing">续期</el-button>
            <!-- 撤销按钮：点击后会弹出二次确认对话框 -->
            <el-button type="danger" size="small" @click="handleRevoke(row)">撤销</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页组件 -->
      <!-- v-model:current-page: 双向绑定当前页码 -->
      <!-- v-model:page-size: 双向绑定每页条数 -->
      <!-- :page-sizes: 可选的每页条数列表 -->
      <!-- @size-change / @current-change: 每页条数或页码变化时重新加载数据 -->
      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[5, 10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadCerts"
          @current-change="loadCerts"
        />
      </div>
    </el-card>

    <!-- 申请新证书对话框 -->
    <el-dialog v-model="showRequestDialog" title="申请新证书" width="500">
      <el-form :model="requestForm" label-width="80px">
        <!-- 域名输入框（必填） -->
        <el-form-item label="域名" required>
          <el-input v-model="requestForm.domain" placeholder="example.com" />
        </el-form-item>
        <!-- Webroot 路径输入框（可选，留空使用 standalone 模式） -->
        <el-form-item label="Webroot">
          <el-input v-model="requestForm.webroot" placeholder="/var/www/html (留空使用 standalone 模式)" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRequestDialog = false">取消</el-button>
        <!-- 提交申请按钮 -->
        <el-button type="primary" @click="handleRequest" :loading="requesting">申请</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
/**
 * 证书列表页面逻辑 (CertificateList.vue)
 * 功能：
 * 1. 加载并展示所有 SSL 证书的分页列表
 * 2. 提供证书续期、撤销操作（含二次确认）
 * 3. 提供申请新证书功能（填写域名和可选 webroot 路径）
 */

// Vue 响应式 API
import { ref, onMounted } from 'vue'
// Element Plus 消息提示和确认弹窗
import { ElMessage, ElMessageBox } from 'element-plus'
// 导入证书相关 API 接口
import { getCertificates, renewCertificate, requestCertificate, revokeCertificate } from '@/api/modules'
// 日期格式化工具
import dayjs from 'dayjs'

// ==================== 响应式状态 ====================

// 证书列表数据
const certs = ref<any[]>([])
// 表格加载状态
const loading = ref(false)
// 证书总数（用于分页组件）
const total = ref(0)
// 当前页码
const page = ref(1)
// 每页显示条数
const pageSize = ref(10)
// 控制申请新证书对话框的显示/隐藏
const showRequestDialog = ref(false)
// 申请按钮的加载状态
const requesting = ref(false)
// 申请新证书表单数据
const requestForm = ref({ domain: '', webroot: '' })

// ==================== 工具函数 ====================

// 格式化日期为 "YYYY-MM-DD" 格式
const formatDate = (d: string) => dayjs(d).format('YYYY-MM-DD')

// 将证书状态英文标识转换为 Element Plus Tag 的 type 属性
const statusType = (s: string) => ({ active: 'success', expiring: 'warning', expired: 'danger' }[s] || 'info') as any

// 将证书状态英文标识转换为中文显示
const statusLabel = (s: string) => ({ active: '正常', expiring: '即将过期', expired: '已过期' }[s] || s)

// ==================== 数据操作函数 ====================

/**
 * 加载证书列表数据（分页）
 * 调用 getCertificates API，传入当前页码和每页条数
 */
const loadCerts = async () => {
  loading.value = true
  try {
    const res = await getCertificates(page.value, pageSize.value)
    certs.value = res.data.items || []  // 更新证书列表
    total.value = res.data.total || 0   // 更新总数
  } catch {
    ElMessage.error('加载证书列表失败')
  } finally {
    loading.value = false
  }
}

/**
 * 处理证书续期操作
 * @param cert - 要续期的证书对象
 * 1. 弹出确认对话框
 * 2. 调用续期 API
 * 3. 成功后刷新列表
 */
const handleRenew = async (cert: any) => {
  try {
    // 弹出确认对话框，显示域名信息
    await ElMessageBox.confirm(`确定要续期证书 ${cert.domain} 吗？`, '确认续期', { type: 'warning' })
    // 设置该行的加载状态（独立于表格全局 loading）
    cert._renewing = true
    // 调用续期接口
    await renewCertificate(cert.id)
    ElMessage.success('续期成功')
    // 刷新证书列表
    loadCerts()
  } catch (e: any) {
    // 用户取消确认（点击"取消"）时不做提示，其他错误显示错误信息
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '续期失败')
  } finally {
    cert._renewing = false
  }
}

/**
 * 处理证书撤销操作
 * @param cert - 要撤销的证书对象
 * 1. 弹出危险确认对话框（强调不可恢复）
 * 2. 调用撤销 API
 * 3. 成功后刷新列表
 */
const handleRevoke = async (cert: any) => {
  try {
    // 使用 error 类型的确认框，强调此操作不可恢复
    await ElMessageBox.confirm(`确定要撤销证书 ${cert.domain} 吗？此操作不可恢复！`, '确认撤销', { type: 'error', confirmButtonText: '确认撤销' })
    await revokeCertificate(cert.id)
    ElMessage.success('证书已撤销')
    loadCerts()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.error || '撤销失败')
  }
}

/**
 * 处理申请新证书操作
 * 1. 验证域名是否填写
 * 2. 调用申请 API
 * 3. 成功后关闭对话框、清空表单、刷新列表
 */
const handleRequest = async () => {
  // 域名为空时提示用户
  if (!requestForm.value.domain) {
    ElMessage.warning('请输入域名')
    return
  }
  requesting.value = true
  try {
    // 调用申请证书接口
    await requestCertificate(requestForm.value.domain, requestForm.value.webroot)
    ElMessage.success('证书申请已提交')
    // 关闭对话框
    showRequestDialog.value = false
    // 清空表单数据
    requestForm.value = { domain: '', webroot: '' }
    // 刷新证书列表
    loadCerts()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '申请失败')
  } finally {
    requesting.value = false
  }
}

// 组件挂载后加载证书列表
onMounted(loadCerts)
</script>

<!-- 证书列表页面样式 -->
<style scoped>
/* 卡片头部：标题和操作按钮分列两端 */
.card-header { display: flex; justify-content: space-between; align-items: center; }
/* 操作按钮组：水平排列，间距 8px */
.actions { display: flex; gap: 8px; }
/* 分页组件容器：右对齐，上方留间距 */
.pagination-wrap { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
