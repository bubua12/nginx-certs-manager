package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

// parsePagination 从 HTTP 请求的查询参数中解析分页信息。
// 如果参数无效或缺失则使用默认值。
// 查询参数:
//   - page: 页码，最小值为 1，默认为 1
//   - page_size: 每页记录数，范围 1-100，默认为 10
//
// 参数:
//   - c: Echo 上下文对象
//
// 返回值:
//   - int: 页码
//   - int: 每页记录数
func parsePagination(c echo.Context) (int, int) {
	// 将查询参数转换为整数，转换失败时默认值为 0
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))

	// 确保页码不小于 1
	if page < 1 {
		page = 1
	}
	// 确保每页记录数在 1-100 之间
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	return page, pageSize
}
