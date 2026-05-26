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

func main() {
	cfg := config.Load()

	// JWT secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "nginx-certs-manager-default-secret-change-me"
	}
	handler.JWTSecret = []byte(jwtSecret)
	middleware.JWTSecret = []byte(jwtSecret)

	if err := database.Init(cfg.DBPath); err != nil {
		log.Fatalf("database init failed: %v", err)
	}

	certbot := service.NewCertbotService(cfg.CertbotDir)
	nginx := service.NewNginxService(cfg.NginxDir)
	scanner := service.NewScanner(certbot, nginx)
	ipLockout := service.NewIPLockout()

	scanner.StartPeriodicScan(30 * time.Minute)

	e := echo.New()
	e.HideBanner = true

	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
	}))
	e.Use(middleware.Logger())

	authHandler := handler.NewAuthHandler(ipLockout)
	certHandler := handler.NewCertHandler(certbot)
	siteHandler := handler.NewSiteHandler(nginx)
	nginxHandler := handler.NewNginxHandler(nginx)

	// Public routes (no auth required)
	public := e.Group("/api")
	{
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/register", authHandler.Register)
	}

	// Protected routes (JWT required)
	api := e.Group("/api", middleware.JWTAuth())
	{
		api.GET("/auth/me", authHandler.GetCurrentUser)
		api.POST("/auth/change-password", authHandler.ChangePassword)

		api.GET("/dashboard/stats", handler.GetDashboardStats)
		api.GET("/dashboard/timeline", handler.GetDashboardTimeline)

		api.GET("/certificates", certHandler.List)
		api.GET("/certificates/:id", certHandler.Get)
		api.POST("/certificates/renew/:id", certHandler.Renew)
		api.POST("/certificates/request", certHandler.RequestCert)
		api.DELETE("/certificates/:id", certHandler.Revoke)

		api.GET("/sites", siteHandler.List)
		api.GET("/sites/:id", siteHandler.Get)
		api.GET("/sites/:id/config", siteHandler.GetConfig)
		api.PUT("/sites/:id/config", siteHandler.UpdateConfig)
		api.POST("/sites/:id/enable", siteHandler.Enable)
		api.POST("/sites/:id/disable", siteHandler.Disable)

		api.GET("/nginx/status", nginxHandler.GetStatus)
		api.POST("/nginx/reload", nginxHandler.Reload)
		api.POST("/nginx/validate", nginxHandler.Validate)

		api.GET("/settings", handler.GetSettings)
		api.PUT("/settings", handler.UpdateSettings)
		api.GET("/logs", handler.GetLogs)
	}

	setupStaticFiles(e, cfg.WebDir)

	log.Printf("Server starting on http://0.0.0.0:%s", cfg.Port)
	e.Logger.Fatal(e.Start("0.0.0.0:" + cfg.Port))
}

func setupStaticFiles(e *echo.Echo, webDir string) {
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		log.Printf("Web directory %s not found, frontend disabled", webDir)
		e.GET("/*", func(c echo.Context) error {
			return c.String(200, "Nginx Certs Manager API is running. Frontend not built yet.")
		})
		return
	}

	e.Static("/assets", webDir+"/assets")
	e.File("/favicon.ico", webDir+"/favicon.ico")

	e.GET("/*", func(c echo.Context) error {
		return c.File(webDir + "/index.html")
	})
}
