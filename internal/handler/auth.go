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

var JWTSecret []byte

type AuthHandler struct {
	ipLockout *service.IPLockout
}

func NewAuthHandler(ipLockout *service.IPLockout) *AuthHandler {
	return &AuthHandler{ipLockout: ipLockout}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token    string   `json:"token"`
	Username string   `json:"username"`
	Role     string   `json:"role"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	ip := clientIP(c)

	// Check IP lockout
	if locked, _ := h.ipLockout.Check(ip); locked {
		remaining := h.ipLockout.GetLockRemaining(ip)
		return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
			"error":    "登录失败次数过多，请稍后再试",
			"retry_after": int(remaining.Minutes()) + 1,
		})
	}

	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "用户名和密码不能为空"})
	}

	var user model.User
	if database.DB.Where("username = ?", req.Username).First(&user).Error != nil {
		h.ipLockout.RecordFailure(ip)
		_, remaining := h.ipLockout.Check(ip)
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":     "用户名或密码错误",
			"remaining": remaining,
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.ipLockout.RecordFailure(ip)
		_, remaining := h.ipLockout.Check(ip)
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":     "用户名或密码错误",
			"remaining": remaining,
		})
	}

	// Success - reset IP lockout
	h.ipLockout.Reset(ip)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenStr, err := token.SignedString(JWTSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "生成令牌失败"})
	}

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

func (h *AuthHandler) Register(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "用户名和密码不能为空"})
	}

	if len(req.Password) < 6 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "密码长度不能少于6位"})
	}

	// Check if user exists
	var count int64
	database.DB.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		return c.JSON(http.StatusConflict, map[string]string{"error": "用户名已存在"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "密码加密失败"})
	}

	// First user is admin
	role := "user"
	var totalUsers int64
	database.DB.Model(&model.User{}).Count(&totalUsers)
	if totalUsers == 0 {
		role = "admin"
	}

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

func (h *AuthHandler) GetCurrentUser(c echo.Context) error {
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

func (h *AuthHandler) ChangePassword(c echo.Context) error {
	userID, _ := c.Get("user_id").(uint)

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
	}

	if len(req.NewPassword) < 6 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "新密码长度不能少于6位"})
	}

	var user model.User
	if database.DB.First(&user, userID).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "用户不存在"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "原密码错误"})
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	database.DB.Model(&user).Update("password", string(hash))

	return c.JSON(http.StatusOK, map[string]string{"message": "密码修改成功"})
}

func clientIP(c echo.Context) string {
	ip := c.Request().Header.Get("X-Real-IP")
	if ip == "" {
		ip = c.Request().Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = strings.Split(c.Request().RemoteAddr, ":")[0]
	}
	return ip
}
