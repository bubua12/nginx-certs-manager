package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
	"nginx-certs-manager/internal/service"
)

// SiteHandler 结构体是站点管理相关的 HTTP 处理器。
// 封装了 Nginx 服务实例，用于执行站点配置的读取、更新、启用和禁用等操作。
type SiteHandler struct {
	nginx *service.NginxService // Nginx 服务实例，用于操作 Nginx 配置文件
}

// NewSiteHandler 创建一个新的站点处理器实例。
// 参数:
//   - nginx: Nginx 服务实例指针
//
// 返回值:
//   - *SiteHandler: 站点处理器实例
func NewSiteHandler(nginx *service.NginxService) *SiteHandler {
	return &SiteHandler{nginx: nginx}
}

// List 处理 GET /api/sites 请求。
// 获取站点列表，支持分页查询，同时预加载关联的证书信息。
// 查询参数: page（页码，默认1）、page_size（每页数量，默认10，最大100）
// 返回值: JSON 对象包含 items（站点列表）、total（总数）、page（当前页）、page_size（每页数量）
func (h *SiteHandler) List(c echo.Context) error {
	page, pageSize := parsePagination(c)

	// 查询站点总数
	var total int64
	database.DB.Model(&model.Site{}).Count(&total)

	// 查询当前页的站点数据，使用 Preload 预加载关联的证书信息（避免 N+1 查询问题）
	var sites []model.Site
	database.DB.Preload("Certificate").Offset((page - 1) * pageSize).Limit(pageSize).Find(&sites)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items":     sites,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 处理 GET /api/sites/:id 请求。
// 根据站点 ID 获取单个站点的详细信息，包含关联的证书数据。
// 路径参数: id（站点 ID）
// 返回值: JSON 格式的站点对象（含证书），或 400/404 错误响应
func (h *SiteHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 查询站点记录，预加载关联的证书信息
	var site model.Site
	if database.DB.Preload("Certificate").First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	return c.JSON(http.StatusOK, site)
}

// GetConfig 处理 GET /api/sites/:id/config 请求。
// 获取指定站点的 Nginx 配置文件内容（server block 原始文本）。
// 路径参数: id（站点 ID）
// 返回值: JSON 包含 content 字段（配置文件内容），或 400/404/500 错误响应
func (h *SiteHandler) GetConfig(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 查询站点记录获取域名
	var site model.Site
	if database.DB.First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	// 从 Nginx 配置文件中读取该域名对应的 server block 内容
	content, err := h.nginx.GetSiteConfig(site.Domain)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"content": content})
}

// UpdateConfig 处理 PUT /api/sites/:id/config 请求。
// 更新指定站点的 Nginx 配置文件内容，并记录操作日志。
// 路径参数: id（站点 ID）
// 请求体参数: content（新的配置内容）
// 返回值: JSON 包含 message，或 400/404/500 错误响应
func (h *SiteHandler) UpdateConfig(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 查询站点记录
	var site model.Site
	if database.DB.First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	// 绑定请求参数
	var req struct {
		Content string `json:"content"` // 新的 Nginx 配置内容
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// 保存站点配置到 Nginx 配置文件
	if err := h.nginx.SaveSiteConfig(site.Domain, req.Content); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 记录配置更新操作日志
	database.DB.Create(&model.OperationLog{
		Type:    "site_config_update",
		Target:  site.Domain,
		Status:  "success",
		Message: "configuration updated",
	})

	return c.JSON(http.StatusOK, map[string]string{"message": "config updated"})
}

// Enable 处理 POST /api/sites/:id/enable 请求。
// 启用指定站点（创建 sites-enabled 符号链接），并更新数据库状态和操作日志。
// 路径参数: id（站点 ID）
// 返回值: JSON 包含 message，或 400/404/500 错误响应
func (h *SiteHandler) Enable(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 查询站点记录
	var site model.Site
	if database.DB.First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	// 调用 Nginx 服务启用站点
	if err := h.nginx.EnableSite(site.Domain); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 更新数据库中站点的启用状态
	database.DB.Model(&site).Update("enabled", true)

	// 记录站点启用操作日志
	database.DB.Create(&model.OperationLog{
		Type:    "site_enable",
		Target:  site.Domain,
		Status:  "success",
		Message: "site enabled",
	})

	return c.JSON(http.StatusOK, map[string]string{"message": "site enabled"})
}

// Disable 处理 POST /api/sites/:id/disable 请求。
// 禁用指定站点（删除 sites-enabled 符号链接），并更新数据库状态和操作日志。
// 路径参数: id（站点 ID）
// 返回值: JSON 包含 message，或 400/404/500 错误响应
func (h *SiteHandler) Disable(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 查询站点记录
	var site model.Site
	if database.DB.First(&site, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "site not found"})
	}

	// 调用 Nginx 服务禁用站点
	if err := h.nginx.DisableSite(site.Domain); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 更新数据库中站点的启用状态
	database.DB.Model(&site).Update("enabled", false)

	// 记录站点禁用操作日志
	database.DB.Create(&model.OperationLog{
		Type:    "site_disable",
		Target:  site.Domain,
		Status:  "success",
		Message: "site disabled",
	})

	return c.JSON(http.StatusOK, map[string]string{"message": "site disabled"})
}
