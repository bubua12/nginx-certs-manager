package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
	"nginx-certs-manager/internal/service"
)

type SiteHandler struct {
	nginx *service.NginxService
}

func NewSiteHandler(nginx *service.NginxService) *SiteHandler {
	return &SiteHandler{nginx: nginx}
}

func (h *SiteHandler) List(c echo.Context) error {
	var sites []model.Site
	database.DB.Preload("Certificate").Find(&sites)
	return c.JSON(http.StatusOK, sites)
}

func (h *SiteHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var site model.Site
	if database.DB.Preload("Certificate").First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	return c.JSON(http.StatusOK, site)
}

func (h *SiteHandler) GetConfig(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var site model.Site
	if database.DB.First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	content, err := h.nginx.GetSiteConfig(site.Domain)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"content": content})
}

func (h *SiteHandler) UpdateConfig(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var site model.Site
	if database.DB.First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if err := h.nginx.SaveSiteConfig(site.Domain, req.Content); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	database.DB.Create(&model.OperationLog{
		Type:    "site_config_update",
		Target:  site.Domain,
		Status:  "success",
		Message: "configuration updated",
	})

	return c.JSON(http.StatusOK, map[string]string{"message": "config updated"})
}

func (h *SiteHandler) Enable(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var site model.Site
	if database.DB.First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	if err := h.nginx.EnableSite(site.Domain); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	database.DB.Model(&site).Update("enabled", true)

	database.DB.Create(&model.OperationLog{
		Type:    "site_enable",
		Target:  site.Domain,
		Status:  "success",
		Message: "site enabled",
	})

	return c.JSON(http.StatusOK, map[string]string{"message": "site enabled"})
}

func (h *SiteHandler) Disable(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var site model.Site
	if database.DB.First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	if err := h.nginx.DisableSite(site.Domain); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	database.DB.Model(&site).Update("enabled", false)

	database.DB.Create(&model.OperationLog{
		Type:    "site_disable",
		Target:  site.Domain,
		Status:  "success",
		Message: "site disabled",
	})

	return c.JSON(http.StatusOK, map[string]string{"message": "site disabled"})
}
