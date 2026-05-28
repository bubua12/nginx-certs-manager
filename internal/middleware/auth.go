package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// JWTSecret 是 JWT 令牌验证所用的密钥。
// 在 main 函数启动时从环境变量加载并设置，与 handler 包中的 JWTSecret 保持一致。
var JWTSecret []byte

// JWTAuth 返回一个 JWT 身份认证中间件函数。
// 该中间件从 HTTP 请求的 Authorization 头中提取 Bearer 令牌，验证其有效性，
// 并将解析出的用户信息（user_id、username、role）设置到 Echo 上下文中，
// 供后续处理器使用。
// 认证失败时返回 401 未授权错误。
// 返回值:
//   - echo.MiddlewareFunc: Echo 中间件函数
func JWTAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 从请求头获取 Authorization 值
			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "未登录"})
			}

			// 提取 Bearer 令牌（去除 "Bearer " 前缀）
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			if tokenStr == auth {
				// 前缀不存在，格式不正确
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "令牌格式错误"})
			}

			// 解析并验证 JWT 令牌
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return JWTSecret, nil // 返回签名密钥用于验证
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "令牌无效或已过期"})
			}

			// 从令牌中提取声明（claims）
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "令牌解析失败"})
			}

			// 将用户信息设置到 Echo 上下文中，供后续处理器使用
			// 注意：JWT 中的数值类型默认为 float64，需要转换为 uint
			c.Set("user_id", uint(claims["user_id"].(float64)))
			c.Set("username", claims["username"].(string))
			c.Set("role", claims["role"].(string))

			// 认证通过，继续执行后续处理器
			return next(c)
		}
	}
}
