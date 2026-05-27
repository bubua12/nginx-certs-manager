package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
)

type DashboardStats struct {
	TotalCerts   int64 `json:"total_certs"`
	ActiveCerts  int64 `json:"active_certs"`
	ExpiringSoon int64 `json:"expiring_soon"`
	ExpiredCerts int64 `json:"expired_certs"`
	TotalSites   int64 `json:"total_sites"`
	ActiveSites  int64 `json:"active_sites"`
	SSLSites     int64 `json:"ssl_sites"`
}

type TimelineItem struct {
	Domain   string `json:"domain"`
	NotAfter string `json:"not_after"`
	DaysLeft int    `json:"days_left"`
	Status   string `json:"status"`
}

func GetDashboardStats(c echo.Context) error {
	var stats DashboardStats

	database.DB.Model(&model.Certificate{}).Count(&stats.TotalCerts)
	database.DB.Model(&model.Certificate{}).Where("status = ?", "active").Count(&stats.ActiveCerts)
	database.DB.Model(&model.Certificate{}).Where("status = ?", "expiring").Count(&stats.ExpiringSoon)
	database.DB.Model(&model.Certificate{}).Where("status = ?", "expired").Count(&stats.ExpiredCerts)
	database.DB.Model(&model.Site{}).Count(&stats.TotalSites)
	database.DB.Model(&model.Site{}).Where("enabled = ?", true).Count(&stats.ActiveSites)
	database.DB.Model(&model.Site{}).Where("ssl_enabled = ?", true).Count(&stats.SSLSites)

	return c.JSON(http.StatusOK, stats)
}

func GetDashboardTimeline(c echo.Context) error {
	var certs []model.Certificate
	database.DB.Order("not_after ASC").Find(&certs)

	items := make([]TimelineItem, 0)
	for _, cert := range certs {
		items = append(items, TimelineItem{
			Domain:   cert.Domain,
			NotAfter: cert.NotAfter.Format("2006-01-02"),
			DaysLeft: cert.DaysUntilExpiry(),
			Status:   cert.Status,
		})
	}

	return c.JSON(http.StatusOK, items)
}
