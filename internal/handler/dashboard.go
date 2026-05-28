// Package handler 提供 HTTP 请求处理函数（控制器层）。
// 每个处理器负责处理特定功能模块的 HTTP 请求，包括参数解析、业务逻辑调用和响应返回。
package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
)

// DashboardStats 结构体表示仪表盘的统计数据。
// 用于汇总展示证书和站点的整体状态信息。
type DashboardStats struct {
	TotalCerts   int64 `json:"total_certs"`   // 证书总数
	ActiveCerts  int64 `json:"active_certs"`  // 状态正常的证书数量
	ExpiringSoon int64 `json:"expiring_soon"` // 即将过期的证书数量（30 天内）
	ExpiredCerts int64 `json:"expired_certs"` // 已过期的证书数量
	TotalSites   int64 `json:"total_sites"`   // 站点总数
	ActiveSites  int64 `json:"active_sites"`  // 已启用的站点数量
	SSLSites     int64 `json:"ssl_sites"`     // 启用 SSL 的站点数量
}

// TimelineItem 结构体表示证书时间线中的一个条目。
// 用于在前端展示证书到期时间线图表。
type TimelineItem struct {
	Domain   string `json:"domain"`    // 证书绑定的域名
	NotAfter string `json:"not_after"` // 证书过期日期，格式 "2006-01-02"
	DaysLeft int    `json:"days_left"` // 距离过期还有多少天（负数表示已过期）
	Status   string `json:"status"`    // 证书状态：active/expiring/expired
}

// GetDashboardStats 处理 GET /api/dashboard/stats 请求。
// 从数据库中统计证书和站点的各种状态数量，返回仪表盘概览数据。
// 不需要请求参数。
// 返回值: JSON 格式的 DashboardStats 统计数据
func GetDashboardStats(c echo.Context) error {
	var stats DashboardStats

	// 统计各类证书数量
	database.DB.Model(&model.Certificate{}).Count(&stats.TotalCerts)                        // 总数
	database.DB.Model(&model.Certificate{}).Where("status = ?", "active").Count(&stats.ActiveCerts)   // 正常证书
	database.DB.Model(&model.Certificate{}).Where("status = ?", "expiring").Count(&stats.ExpiringSoon) // 即将过期
	database.DB.Model(&model.Certificate{}).Where("status = ?", "expired").Count(&stats.ExpiredCerts)  // 已过期

	// 统计各类站点数量
	database.DB.Model(&model.Site{}).Count(&stats.TotalSites)                            // 总数
	database.DB.Model(&model.Site{}).Where("enabled = ?", true).Count(&stats.ActiveSites)  // 已启用
	database.DB.Model(&model.Site{}).Where("ssl_enabled = ?", true).Count(&stats.SSLSites) // 启用 SSL

	return c.JSON(http.StatusOK, stats)
}

// GetDashboardTimeline 处理 GET /api/dashboard/timeline 请求。
// 获取所有证书的到期时间信息，按过期日期升序排列，用于前端时间线图表展示。
// 不需要请求参数。
// 返回值: JSON 数组格式的 TimelineItem 列表
func GetDashboardTimeline(c echo.Context) error {
	// 查询所有证书，按过期日期升序排列（最先过期的排在前面）
	var certs []model.Certificate
	database.DB.Order("not_after ASC").Find(&certs)

	// 将证书记录转换为时间线条目格式
	items := make([]TimelineItem, 0)
	for _, cert := range certs {
		items = append(items, TimelineItem{
			Domain:   cert.Domain,
			NotAfter: cert.NotAfter.Format("2006-01-02"), // 使用 Go 时间格式化标准
			DaysLeft: cert.DaysUntilExpiry(),
			Status:   cert.Status,
		})
	}

	return c.JSON(http.StatusOK, items)
}
