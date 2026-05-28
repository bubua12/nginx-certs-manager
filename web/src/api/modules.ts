/**
 * API 接口封装模块 (api/modules.ts)
 * 将后端所有 RESTful API 按功能分类封装为独立函数
 * 供 Vue 组件直接调用，实现前后端数据交互
 */

// 导入配置好的 Axios 实例
import api from './index'

// ==================== 仪表盘相关接口 ====================

// 获取仪表盘统计数据（证书总数、正常/过期数量、站点数量等）
export const getDashboardStats = () => api.get('/dashboard/stats')

// 获取证书到期时间线数据（用于仪表盘图表展示）
export const getDashboardTimeline = () => api.get('/dashboard/timeline')

// ==================== 证书管理相关接口 ====================

/**
 * 获取证书列表（分页）
 * @param page - 当前页码，默认第 1 页
 * @param pageSize - 每页条数，默认 10 条
 */
export const getCertificates = (page = 1, pageSize = 10) =>
  api.get('/certificates', { params: { page, page_size: pageSize } })

/**
 * 获取单个证书详情
 * @param id - 证书 ID
 */
export const getCertificate = (id: number) => api.get(`/certificates/${id}`)

/**
 * 续期指定证书
 * @param id - 证书 ID
 */
export const renewCertificate = (id: number) => api.post(`/certificates/renew/${id}`)

/**
 * 申请新证书
 * @param domain - 要申请证书的域名（如 example.com）
 * @param webroot - 可选的 webroot 路径，用于 HTTP 验证模式；留空则使用 standalone 模式
 */
export const requestCertificate = (domain: string, webroot?: string) =>
  api.post('/certificates/request', { domain, webroot })

/**
 * 撤销证书（不可恢复）
 * @param id - 证书 ID
 */
export const revokeCertificate = (id: number) => api.delete(`/certificates/${id}`)

// ==================== 站点管理相关接口 ====================

/**
 * 获取站点列表（分页）
 * @param page - 当前页码，默认第 1 页
 * @param pageSize - 每页条数，默认 10 条
 */
export const getSites = (page = 1, pageSize = 10) =>
  api.get('/sites', { params: { page, page_size: pageSize } })

/**
 * 获取单个站点详情
 * @param id - 站点 ID
 */
export const getSite = (id: number) => api.get(`/sites/${id}`)

/**
 * 获取站点的 Nginx 配置文件内容
 * @param id - 站点 ID
 */
export const getSiteConfig = (id: number) => api.get(`/sites/${id}/config`)

/**
 * 更新站点的 Nginx 配置文件内容
 * @param id - 站点 ID
 * @param content - 新的配置文件文本内容
 */
export const updateSiteConfig = (id: number, content: string) =>
  api.put(`/sites/${id}/config`, { content })

/**
 * 启用站点（在 Nginx 中激活该站点配置）
 * @param id - 站点 ID
 */
export const enableSite = (id: number) => api.post(`/sites/${id}/enable`)

/**
 * 禁用站点（在 Nginx 中停用该站点配置）
 * @param id - 站点 ID
 */
export const disableSite = (id: number) => api.post(`/sites/${id}/disable`)

// ==================== Nginx 管理相关接口 ====================

// 获取 Nginx 运行状态（运行/停止、版本号、PID 等）
export const getNginxStatus = () => api.get('/nginx/status')

// 重新加载 Nginx 配置（不中断服务的平滑重载）
export const reloadNginx = () => api.post('/nginx/reload')

// 校验 Nginx 配置文件语法是否正确
export const validateNginx = () => api.post('/nginx/validate')

// ==================== 系统设置与日志相关接口 ====================

// 获取系统设置（扫描间隔、通知方式、预警天数等）
export const getSettings = () => api.get('/settings')

/**
 * 更新系统设置
 * @param settings - 设置键值对，键和值均为字符串
 */
export const updateSettings = (settings: Record<string, string>) =>
  api.put('/settings', settings)

/**
 * 获取操作日志列表（分页）
 * @param page - 当前页码，默认第 1 页
 * @param pageSize - 每页条数，默认 10 条
 */
export const getLogs = (page = 1, pageSize = 10) =>
  api.get('/logs', { params: { page, page_size: pageSize } })
