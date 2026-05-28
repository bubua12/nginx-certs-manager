// Package middleware 提供 HTTP 中间件功能。
// 中间件在请求到达处理器之前或响应返回客户端之后执行通用逻辑，
// 如请求日志记录、身份认证验证等。
package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
)

// Logger 返回一个请求日志记录中间件函数。
// 该中间件记录每个 HTTP 请求的方法、路径、状态码和处理耗时。
// 日志格式示例: "GET /api/certificates 200 15.2ms"
// 返回值:
//   - echo.MiddlewareFunc: Echo 中间件函数
func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 记录请求开始时间
			start := time.Now()
			// 执行后续处理器
			err := next(c)
			// 记录请求日志：HTTP 方法、请求路径、响应状态码、处理耗时
			c.Logger().Infof("%s %s %d %s",
				c.Request().Method,
				c.Request().URL.Path,
				c.Response().Status,
				time.Since(start),
			)
			return err
		}
	}
}
