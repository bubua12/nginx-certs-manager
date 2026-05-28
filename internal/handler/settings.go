package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
)

// GetSettings 处理 GET /api/settings 请求。
// 获取系统所有设置项，以键值对的形式返回。
// 不需要请求参数。
// 返回值: JSON 对象，键为设置项名称，值为设置项值
func GetSettings(c echo.Context) error {
	// 从数据库查询所有设置项
	var settings []model.Setting
	database.DB.Find(&settings)

	// 将设置列表转换为键值对映射，方便前端使用
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	return c.JSON(http.StatusOK, result)
}

// UpdateSettings 处理 PUT /api/settings 请求。
// 批量更新系统设置项，使用 GORM 的 Save 方法实现 upsert（存在则更新，不存在则创建）。
// 请求体: JSON 对象，键为设置项名称，值为设置项值
// 返回值: JSON 包含 message，或 400 错误响应
func UpdateSettings(c echo.Context) error {
	// 绑定请求参数（键值对映射）
	var req map[string]string
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// 遍历请求中的每个设置项，逐个保存到数据库
	// 使用 GORM 的 Save 方法会自动根据主键判断是插入还是更新
	for key, value := range req {
		setting := model.Setting{Key: key, Value: value}
		database.DB.Save(&setting)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "settings updated"})
}

// GetLogs 处理 GET /api/logs 请求。
// 获取系统操作日志列表，支持分页查询，结果按创建时间降序排列（最新的在前面）。
// 查询参数: page（页码，默认1）、page_size（每页数量，默认10，最大100）
// 返回值: JSON 对象包含 items（日志列表）、total（总数）、page（当前页）、page_size（每页数量）
func GetLogs(c echo.Context) error {
	// 解析分页参数
	page, pageSize := parsePagination(c)

	// 查询日志总数
	var total int64
	database.DB.Model(&model.OperationLog{}).Count(&total)

	// 查询当前页的日志数据，按创建时间降序排列
	var logs []model.OperationLog
	database.DB.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items":     logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
