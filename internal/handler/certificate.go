package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
	"nginx-certs-manager/internal/service"
)

// CertHandler 结构体是证书管理相关的 HTTP 处理器。
// 封装了 Certbot 服务实例，用于执行证书的申请、续期和撤销等操作。
type CertHandler struct {
	certbot *service.CertbotService // Certbot 服务实例，用于与 certbot 命令行工具交互
}

// NewCertHandler 创建一个新的证书处理器实例。
// 参数:
//   - certbot: Certbot 服务实例指针
//
// 返回值:
//   - *CertHandler: 证书处理器实例
func NewCertHandler(certbot *service.CertbotService) *CertHandler {
	return &CertHandler{certbot: certbot}
}

// List 处理 GET /api/certificates 请求。
// 获取证书列表，支持分页查询，结果按过期时间升序排列（最先过期的排在前面）。
// 查询参数: page（页码，默认1）、page_size（每页数量，默认10，最大100）
// 返回值: JSON 对象包含 items（证书列表）、total（总数）、page（当前页）、page_size（每页数量）
// 每个证书项额外包含 days_left 字段（距离过期的天数）。
func (h *CertHandler) List(c echo.Context) error {
	// 解析分页参数
	page, pageSize := parsePagination(c)

	// 查询证书总数
	var total int64
	database.DB.Model(&model.Certificate{}).Count(&total)

	// 查询当前页的证书数据，按过期日期升序排列
	var certs []model.Certificate
	database.DB.Order("not_after ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&certs)

	// 定义证书视图结构体，在原始证书信息基础上增加天数字段
	type CertView struct {
		model.Certificate
		DaysLeft int `json:"days_left"` // 距离过期还有多少天
	}

	// 将证书记录转换为视图格式，计算剩余天数
	result := make([]CertView, 0)
	for _, cert := range certs {
		result = append(result, CertView{
			Certificate: cert,
			DaysLeft:    cert.DaysUntilExpiry(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items":     result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 处理 GET /api/certificates/:id 请求。
// 根据证书 ID 获取单个证书的详细信息。
// 路径参数: id（证书 ID）
// 返回值: JSON 格式的证书对象，或 400/404 错误响应
func (h *CertHandler) Get(c echo.Context) error {
	// 解析路径参数中的证书 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 从数据库查询证书记录
	var cert model.Certificate
	if database.DB.First(&cert, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "certificate not found"})
	}

	return c.JSON(http.StatusOK, cert)
}

// Renew 处理 POST /api/certificates/renew/:id 请求。
// 续期指定证书，调用 Certbot 执行续期操作，并记录操作日志。
// 路径参数: id（证书 ID）
// 返回值: JSON 包含 message 和 certbot 输出信息，或 400/404/500 错误响应
func (h *CertHandler) Renew(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 查询证书记录
	var cert model.Certificate
	if database.DB.First(&cert, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "certificate not found"})
	}

	// 调用 Certbot 执行证书续期
	output, err := h.certbot.Renew(cert.Domain)

	// 创建操作日志条目
	logEntry := model.OperationLog{
		Type:   "cert_renew",
		Target: cert.Domain,
		Status: "success",
	}
	if err != nil {
		// 续期失败，记录错误信息到日志
		logEntry.Status = "failed"
		logEntry.Message = err.Error() + "\n + output"
		database.DB.Create(&logEntry)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error(), "output": output})
	}

	// 续期成功，记录输出信息并更新证书状态
	logEntry.Message = output
	database.DB.Create(&logEntry)

	// 更新数据库中证书的状态和最后续期时间
	database.DB.Model(&cert).Updates(map[string]interface{}{
		"status":       "active",
		"last_renewed": database.DB.NowFunc(),
	})

	return c.JSON(http.StatusOK, map[string]string{"message": "renewal successful", "output": output})
}

// RequestCert 处理 POST /api/certificates/request 请求。
// 申请新的 SSL 证书，调用 Certbot 执行证书申请操作。
// 请求体参数: domain（域名，必填）、webroot（Webroot 路径，可选）
// 如果提供了 webroot 则使用 webroot 验证方式，否则使用 standalone 方式。
// 返回值: JSON 包含 message 和 certbot 输出信息，或 400/500 错误响应
func (h *CertHandler) RequestCert(c echo.Context) error {
	// 绑定请求参数
	var req struct {
		Domain  string `json:"domain"`  // 要申请证书的域名
		Webroot string `json:"webroot"` // Webroot 验证路径（可选）
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Domain == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "domain is required"})
	}

	// 调用 Certbot 申请证书
	output, err := h.certbot.RequestCert(req.Domain, req.Webroot)

	// 创建操作日志条目
	logEntry := model.OperationLog{
		Type:   "cert_request",
		Target: req.Domain,
		Status: "success",
	}
	if err != nil {
		// 申请失败，记录错误信息
		logEntry.Status = "failed"
		logEntry.Message = err.Error() + "\n" + output
		database.DB.Create(&logEntry)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error(), "output": output})
	}

	// 申请成功，记录输出信息
	logEntry.Message = output
	database.DB.Create(&logEntry)

	return c.JSON(http.StatusOK, map[string]string{"message": "certificate requested", "output": output})
}

// Revoke 处理 DELETE /api/certificates/:id 请求。
// 撤销指定证书，调用 Certbot 执行撤销操作，并从数据库中删除证书记录。
// 路径参数: id（证书 ID）
// 返回值: JSON 包含 message，或 400/404/500 错误响应
func (h *CertHandler) Revoke(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	// 查询证书记录
	var cert model.Certificate
	if database.DB.First(&cert, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "certificate not found"})
	}

	// 调用 Certbot 撤销证书（传入证书文件路径）
	output, err := h.certbot.Revoke(cert.CertPath)

	// 创建操作日志条目
	logEntry := model.OperationLog{
		Type:   "cert_revoke",
		Target: cert.Domain,
		Status: "success",
	}
	if err != nil {
		// 撤销失败，记录错误信息
		logEntry.Status = "failed"
		logEntry.Message = err.Error() + "\n" + output
		database.DB.Create(&logEntry)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error(), "output": output})
	}

	// 撤销成功，记录输出信息
	logEntry.Message = output
	database.DB.Create(&logEntry)

	// 从数据库中删除证书记录
	database.DB.Delete(&cert)

	return c.JSON(http.StatusOK, map[string]string{"message": "certificate revoked"})
}
