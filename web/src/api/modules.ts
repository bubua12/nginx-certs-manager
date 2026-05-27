import api from './index'

// Dashboard
export const getDashboardStats = () => api.get('/dashboard/stats')
export const getDashboardTimeline = () => api.get('/dashboard/timeline')

// Certificates
export const getCertificates = (page = 1, pageSize = 10) =>
  api.get('/certificates', { params: { page, page_size: pageSize } })
export const getCertificate = (id: number) => api.get(`/certificates/${id}`)
export const renewCertificate = (id: number) => api.post(`/certificates/renew/${id}`)
export const requestCertificate = (domain: string, webroot?: string) =>
  api.post('/certificates/request', { domain, webroot })
export const revokeCertificate = (id: number) => api.delete(`/certificates/${id}`)

// Sites
export const getSites = (page = 1, pageSize = 10) =>
  api.get('/sites', { params: { page, page_size: pageSize } })
export const getSite = (id: number) => api.get(`/sites/${id}`)
export const getSiteConfig = (id: number) => api.get(`/sites/${id}/config`)
export const updateSiteConfig = (id: number, content: string) =>
  api.put(`/sites/${id}/config`, { content })
export const enableSite = (id: number) => api.post(`/sites/${id}/enable`)
export const disableSite = (id: number) => api.post(`/sites/${id}/disable`)

// Nginx
export const getNginxStatus = () => api.get('/nginx/status')
export const reloadNginx = () => api.post('/nginx/reload')
export const validateNginx = () => api.post('/nginx/validate')

// Settings & Logs
export const getSettings = () => api.get('/settings')
export const updateSettings = (settings: Record<string, string>) =>
  api.put('/settings', settings)
export const getLogs = (page = 1, pageSize = 10) =>
  api.get('/logs', { params: { page, page_size: pageSize } })
