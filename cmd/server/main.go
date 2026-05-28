// Package main 是 nginx-certs-manager 应用程序的入口点。
// 该应用程序是一个 Nginx 证书管理器，提供 Web 界面来管理 SSL 证书和 Nginx 站点配置。
// 主要功能包括：证书申请/续期/撤销、站点管理、Nginx 状态监控、操作日志记录等。
package main

import (
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"

	"nginx-certs-manager/internal/config"
	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/handler"
	"nginx-certs-manager/internal/middleware"
	"nginx-certs-manager/internal/service"
)

// main 函数是应用程序的主入口点，负责以下初始化工作：
// 1. 加载配置（从环境变量读取）
// 2. 设置 JWT 密钥（用于用户认证令牌的签发和验证）
// 3. 初始化数据库连接并执行自动迁移
// 4. 创建各种服务实例（Certbot、Nginx、扫描器、IP 锁定）
// 5. 启动定时证书扫描任务
// 6. 配置 HTTP 路由和中间件
// 7. 配置静态文件服务（前端资源）
// 8. 启动 HTTP 服务器监听
func main() {
	// 从环境变量加载配置，包括端口号、数据库路径、Web 目录、Nginx 配置目录等
	cfg := config.Load()

	// 从环境变量读取 JWT 密钥，如果未设置则使用默认值（生产环境应更换）
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "nginx-certs-manager-default-secret-change-me"
	}
	// 将 JWT 密钥设置到 handler 和 middleware 包的全局变量中，供认证使用
	handler.JWTSecret = []byte(jwtSecret)
	middleware.JWTSecret = []byte(jwtSecret)

	// 初始化数据库，包括创建表结构和默认管理员账户
	if err := database.Init(cfg.DBPath); err != nil {
		log.Fatalf("database init failed: %v", err)
	}

	// 创建 Certbot 服务实例，用于 SSL 证书的申请、续期和撤销操作
	certbot := service.NewCertbotService(cfg.CertbotDir)
	// 创建 Nginx 服务实例，用于管理 Nginx 站点配置、状态查询和重载操作
	nginx := service.NewNginxService(cfg.NginxDir)
	// 创建扫描器实例，用于定期扫描本地证书文件和 Nginx 站点配置并同步到数据库
	scanner := service.NewScanner(certbot, nginx)
	// 创建 IP 锁定服务实例，用于防止暴力破解登录密码
	ipLockout := service.NewIPLockout()

	// 启动定时扫描任务，每 30 分钟执行一次全量扫描（证书 + 站点）
	scanner.StartPeriodicScan(30 * time.Minute)

	// 创建 Echo Web 框架实例并隐藏启动横幅
	e := echo.New()
	e.HideBanner = true

	// 注册全局中间件：
	// Recover - 捕获 panic 并恢复，防止服务器崩溃
	// CORS - 配置跨域资源共享，允许所有来源访问 API
	// Logger - 自定义请求日志中间件，记录请求方法、路径、状态码和耗时
	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
	}))
	e.Use(middleware.Logger())

	// 创建各功能模块的处理器实例
	authHandler := handler.NewAuthHandler(ipLockout)  // 认证处理器（登录、注册、密码修改）
	certHandler := handler.NewCertHandler(certbot)     // 证书处理器（列表、详情、续期、申请、撤销）
	siteHandler := handler.NewSiteHandler(nginx)       // 站点处理器（列表、详情、配置、启用/禁用）
	nginxHandler := handler.NewNginxHandler(nginx)     // Nginx 处理器（状态、重载、验证）

	// 公开路由组（不需要认证即可访问）
	public := e.Group("/api")
	{
		public.POST("/auth/login", authHandler.Login) // 用户登录接口
	}

	// 受保护路由组（需要 JWT 令牌认证才能访问）
	api := e.Group("/api", middleware.JWTAuth())
	{
		// 认证相关接口
		api.GET("/auth/me", authHandler.GetCurrentUser)           // 获取当前登录用户信息
		api.POST("/auth/change-password", authHandler.ChangePassword) // 修改密码

		// 仪表盘数据接口
		api.GET("/dashboard/stats", handler.GetDashboardStats)     // 获取统计概览数据
		api.GET("/dashboard/timeline", handler.GetDashboardTimeline) // 获取证书到期时间线

		// 证书管理接口
		api.GET("/certificates", certHandler.List)                  // 获取证书列表（分页）
		api.GET("/certificates/:id", certHandler.Get)               // 获取证书详情
		api.POST("/certificates/renew/:id", certHandler.Renew)      // 续期指定证书
		api.POST("/certificates/request", certHandler.RequestCert)  // 申请新证书
		api.DELETE("/certificates/:id", certHandler.Revoke)         // 撤销证书

		// 站点管理接口
		api.GET("/sites", siteHandler.List)                   // 获取站点列表（分页）
		api.GET("/sites/:id", siteHandler.Get)                // 获取站点详情
		api.GET("/sites/:id/config", siteHandler.GetConfig)   // 获取站点 Nginx 配置内容
		api.PUT("/sites/:id/config", siteHandler.UpdateConfig) // 更新站点 Nginx 配置
		api.POST("/sites/:id/enable", siteHandler.Enable)     // 启用站点
		api.POST("/sites/:id/disable", siteHandler.Disable)   // 禁用站点

		// Nginx 管理接口
		api.GET("/nginx/status", nginxHandler.GetStatus)   // 获取 Nginx 运行状态
		api.POST("/nginx/reload", nginxHandler.Reload)     // 重载 Nginx 配置
		api.POST("/nginx/validate", nginxHandler.Validate) // 验证 Nginx 配置语法

		// 系统设置和日志接口
		api.GET("/settings", handler.GetSettings)   // 获取系统设置
		api.PUT("/settings", handler.UpdateSettings) // 更新系统设置
		api.GET("/logs", handler.GetLogs)            // 获取操作日志（分页）
	}

	// 配置静态文件服务（前端 SPA 应用）
	setupStaticFiles(e, cfg.WebDir)

	// 启动 HTTP 服务器，监听指定端口
	log.Printf("Server starting on http://0.0.0.0:%s", cfg.Port)
	e.Logger.Fatal(e.Start("0.0.0.0:" + cfg.Port))
}

// setupStaticFiles 配置前端静态文件服务。
// 如果前端构建目录存在，则提供静态资源（JS/CSS/图片）和 SPA 回退路由；
// 如果目录不存在，则显示提示信息，仅 API 服务可用。
// 参数:
//   - e: Echo 框架实例
//   - webDir: 前端构建产物目录路径
func setupStaticFiles(e *echo.Echo, webDir string) {
	// 检查前端目录是否存在
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		log.Printf("Web directory %s not found, frontend disabled", webDir)
		// 目录不存在时，根路径返回提示信息
		e.GET("/*", func(c echo.Context) error {
			return c.String(200, "Nginx Certs Manager API is running. Frontend not built yet.")
		})
		return
	}

	// 将 /assets 路径映射到前端构建目录中的 assets 子目录（包含 JS、CSS 等）
	e.Static("/assets", webDir+"/assets")
	// 提供 favicon.ico 文件
	e.File("/favicon.ico", webDir+"/favicon.ico")

	// 所有未匹配的路由都返回 index.html，支持前端 SPA 路由
	e.GET("/*", func(c echo.Context) error {
		return c.File(webDir + "/index.html")
	})
}
