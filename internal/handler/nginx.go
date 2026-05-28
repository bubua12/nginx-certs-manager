package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/service"
)

// NginxHandler 结构体是 Nginx 管理相关的 HTTP 处理器。
// 提供 Nginx 服务状态查询、配置重载和配置验证等功能。
type NginxHandler struct {
	nginx *service.NginxService // Nginx 服务实例，用于执行 Nginx 相关操作
}

// NewNginxHandler 创建一个新的 Nginx 处理器实例。
// 参数:
//   - nginx: Nginx 服务实例指针
//
// 返回值:
//   - *NginxHandler: Nginx 处理器实例
func NewNginxHandler(nginx *service.NginxService) *NginxHandler {
	return &NginxHandler{nginx: nginx}
}

// GetStatus 处理 GET /api/nginx/status 请求。
// 获取 Nginx 服务的运行状态，包括是否正在运行、版本号和进程 PID。
// 不需要请求参数。
// 返回值: JSON 格式的 NginxStatus 对象（包含 running、version、pid 字段）
func (h *NginxHandler) GetStatus(c echo.Context) error {
	status, err := h.nginx.GetStatus()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, status)
}

// Reload 处理 POST /api/nginx/reload 请求。
// 执行 nginx -s reload 命令重新加载 Nginx 配置，使配置变更生效。
// 不需要请求参数。
// 返回值: JSON 包含 message 和 nginx 命令输出，或 500 错误响应
func (h *NginxHandler) Reload(c echo.Context) error {
	output, err := h.nginx.Reload()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error(), "output": output})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "nginx reloaded", "output": output})
}

// Validate 处理 POST /api/nginx/validate 请求。
// 执行 nginx -t 命令验证 Nginx 配置文件语法是否正确。
// 不需要请求参数。
// 返回值: JSON 包含 valid（布尔值，是否有效）和 output（验证输出信息）
func (h *NginxHandler) Validate(c echo.Context) error {
	valid, output, err := h.nginx.Validate()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":  valid,  // 配置是否有效
		"output": output, // nginx -t 的输出信息
	})
}
