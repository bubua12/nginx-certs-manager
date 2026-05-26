package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/service"
)

type NginxHandler struct {
	nginx *service.NginxService
}

func NewNginxHandler(nginx *service.NginxService) *NginxHandler {
	return &NginxHandler{nginx: nginx}
}

func (h *NginxHandler) GetStatus(c echo.Context) error {
	status, err := h.nginx.GetStatus()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, status)
}

func (h *NginxHandler) Reload(c echo.Context) error {
	output, err := h.nginx.Reload()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error(), "output": output})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "nginx reloaded", "output": output})
}

func (h *NginxHandler) Validate(c echo.Context) error {
	valid, output, err := h.nginx.Validate()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"valid":  valid,
		"output": output,
	})
}
