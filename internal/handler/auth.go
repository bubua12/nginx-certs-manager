package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
	"nginx-certs-manager/internal/service"
)

// JWTSecret 是 JWT 令牌签名和验证所用的密钥。
// 在 main 函数启动时从环境变量加载并设置。
var JWTSecret []byte

// AuthHandler 结构体是认证相关的 HTTP 处理器。
// 处理用户登录、注册、获取当前用户信息和修改密码等请求。
// 集成了 IP 锁定服务以防止暴力破解攻击。
type AuthHandler struct {
	ipLockout *service.IPLockout // IP 锁定服务，用于限制登录失败次数
}

// NewAuthHandler 创建一个新的认证处理器实例。
// 参数:
//   - ipLockout: IP 锁定服务实例指针
//
// 返回值:
//   - *AuthHandler: 认证处理器实例
func NewAuthHandler(ipLockout *service.IPLockout) *AuthHandler {
	return &AuthHandler{ipLockout: ipLockout}
}

// LoginRequest 结构体定义了登录请求的参数格式。
type LoginRequest struct {
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码（明文，传输层应使用 HTTPS）
}

// LoginResponse 结构体定义了登录成功后的响应格式。
type LoginResponse struct {
	Token    string `json:"token"`    // JWT 令牌，后续请求需在 Authorization 头中携带
	Username string `json:"username"` // 用户名
	Role     string `json:"role"`     // 用户角色（admin/user）
}

// Login 处理 POST /api/auth/login 请求。
// 验证用户凭据，成功后返回 JWT 令牌。包含 IP 锁定保护机制：
// 同一 IP 连续登录失败 5 次后将被锁定 30 分钟。
// 请求体参数: username（用户名）、password（密码）
// 返回值: JSON 格式的 LoginResponse，或 400/401/429 错误响应
func (h *AuthHandler) Login(c echo.Context) error {
	// 获取客户端真实 IP 地址
	ip := clientIP(c)

	// 检查该 IP 是否已被锁定（登录失败次数过多）
	if locked, _ := h.ipLockout.Check(ip); locked {
		remaining := h.ipLockout.GetLockRemaining(ip)
		return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
			"error":      "登录失败次数过多，请稍后再试",
			"retry_after": int(remaining.Minutes()) + 1, // 剩余锁定时间（分钟）
		})
	}

	// 绑定请求参数
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
	}

	// 验证用户名和密码不为空
	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "用户名和密码不能为空"})
	}

	// 从数据库查询用户记录
	var user model.User
	if database.DB.Where("username = ?", req.Username).First(&user).Error != nil {
		// 用户不存在，记录失败尝试
		h.ipLockout.RecordFailure(ip)
		_, remaining := h.ipLockout.Check(ip)
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":     "用户名或密码错误", // 不暴露用户是否存在
			"remaining": remaining,        // 剩余尝试次数
		})
	}

	// 使用 bcrypt 验证密码是否匹配
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		// 密码错误，记录失败尝试
		h.ipLockout.RecordFailure(ip)
		_, remaining := h.ipLockout.Check(ip)
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":     "用户名或密码错误",
			"remaining": remaining,
		})
	}

	// 登录成功，重置该 IP 的失败记录
	h.ipLockout.Reset(ip)

	// 生成 JWT 令牌，包含用户信息和过期时间（24 小时）
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,       // 用户 ID
		"username": user.Username, // 用户名
		"role":     user.Role,     // 用户角色
		"exp":      time.Now().Add(24 * time.Hour).Unix(), // 过期时间：24 小时后
	})

	// 使用密钥签名令牌
	tokenStr, err := token.SignedString(JWTSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "生成令牌失败"})
	}

	// 记录登录成功的操作日志
	database.DB.Create(&model.OperationLog{
		Type:    "user_login",
		Target:  user.Username,
		Status:  "success",
		Message: "登录 IP: " + ip,
	})

	return c.JSON(http.StatusOK, LoginResponse{
		Token:    tokenStr,
		Username: user.Username,
		Role:     user.Role,
	})
}

// Register 处理用户注册请求（当前未在路由中注册）。
// 创建新用户账户，密码最少 6 位。第一个注册的用户自动获得 admin 角色。
// 请求体参数: username（用户名）、password（密码）
// 返回值: JSON 包含注册结果信息，或 400/409 错误响应
func (h *AuthHandler) Register(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
	}

	// 验证用户名和密码不为空
	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "用户名和密码不能为空"})
	}

	// 验证密码长度
	if len(req.Password) < 6 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "密码长度不能少于6位"})
	}

	// 检查用户名是否已存在
	var count int64
	database.DB.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		return c.JSON(http.StatusConflict, map[string]string{"error": "用户名已存在"})
	}

	// 使用 bcrypt 对密码进行哈希处理
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "密码加密失败"})
	}

	// 确定用户角色：第一个注册的用户自动成为管理员
	role := "user"
	var totalUsers int64
	database.DB.Model(&model.User{}).Count(&totalUsers)
	if totalUsers == 0 {
		role = "admin" // 系统中没有任何用户时，第一个用户为管理员
	}

	// 创建用户记录
	user := model.User{
		Username: req.Username,
		Password: string(hash),
		Role:     role,
	}
	database.DB.Create(&user)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "注册成功",
		"username": user.Username,
		"role":     user.Role,
	})
}

// GetCurrentUser 处理 GET /api/auth/me 请求。
// 获取当前登录用户的详细信息（需要 JWT 认证）。
// 从 JWT 中间件设置的上下文中获取 user_id。
// 返回值: JSON 包含用户 id、username、role，或 404 错误响应
func (h *AuthHandler) GetCurrentUser(c echo.Context) error {
	// 从 Echo 上下文中获取 JWT 中间件解析出的用户 ID
	userID, _ := c.Get("user_id").(uint)
	var user model.User
	if database.DB.First(&user, userID).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "用户不存在"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

// ChangePassword 处理 POST /api/auth/change-password 请求。
// 修改当前登录用户的密码，需要验证原密码。新密码最少 6 位。
// 请求体参数: old_password（原密码）、new_password（新密码）
// 返回值: JSON 包含 message，或 400/401/404 错误响应
func (h *AuthHandler) ChangePassword(c echo.Context) error {
	// 从上下文中获取当前用户 ID
	userID, _ := c.Get("user_id").(uint)

	// 绑定请求参数
	var req struct {
		OldPassword string `json:"old_password"` // 原密码
		NewPassword string `json:"new_password"` // 新密码
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
	}

	// 验证新密码长度
	if len(req.NewPassword) < 6 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "新密码长度不能少于6位"})
	}

	// 查询用户记录
	var user model.User
	if database.DB.First(&user, userID).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "用户不存在"})
	}

	// 验证原密码是否正确
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "原密码错误"})
	}

	// 对新密码进行哈希处理并更新数据库
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	database.DB.Model(&user).Update("password", string(hash))

	return c.JSON(http.StatusOK, map[string]string{"message": "密码修改成功"})
}

// clientIP 辅助函数：获取客户端的真实 IP 地址。
// 按优先级检查以下来源：
// 1. X-Real-IP 请求头（反向代理设置）
// 2. X-Forwarded-For 请求头（负载均衡器设置，取第一个 IP）
// 3. 连接的 RemoteAddr（直连情况）
// 参数:
//   - c: Echo 上下文
//
// 返回值:
//   - string: 客户端 IP 地址
func clientIP(c echo.Context) string {
	ip := c.Request().Header.Get("X-Real-IP")
	if ip == "" {
		ip = c.Request().Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		// RemoteAddr 格式为 "ip:port"，需要提取 IP 部分
		ip = strings.Split(c.Request().RemoteAddr, ":")[0]
	}
	return ip
}
