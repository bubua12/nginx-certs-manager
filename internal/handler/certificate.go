package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
	"nginx-certs-manager/internal/service"
)

type CertHandler struct {
	certbot *service.CertbotService
}

func NewCertHandler(certbot *service.CertbotService) *CertHandler {
	return &CertHandler{certbot: certbot}
}

func (h *CertHandler) List(c echo.Context) error {
	var certs []model.Certificate
	database.DB.Order("not_after ASC").Find(&certs)

	type CertView struct {
		model.Certificate
		DaysLeft int `json:"days_left"`
	}

	result := make([]CertView, 0)
	for _, cert := range certs {
		result = append(result, CertView{
			Certificate: cert,
			DaysLeft:    cert.DaysUntilExpiry(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *CertHandler) Get(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var cert model.Certificate
	if database.DB.First(&cert, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "certificate not found"})
	}

	return c.JSON(http.StatusOK, cert)
}

func (h *CertHandler) Renew(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var cert model.Certificate
	if database.DB.First(&cert, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "certificate not found"})
	}

	output, err := h.certbot.Renew(cert.Domain)

	logEntry := model.OperationLog{
		Type:   "cert_renew",
		Target: cert.Domain,
		Status: "success",
	}
	if err != nil {
		logEntry.Status = "failed"
		logEntry.Message = err.Error() + "\n" + output
		database.DB.Create(&logEntry)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error(), "output": output})
	}

	logEntry.Message = output
	database.DB.Create(&logEntry)

	database.DB.Model(&cert).Updates(map[string]interface{}{
		"status":        "active",
		"last_renewed":  database.DB.NowFunc(),
	})

	return c.JSON(http.StatusOK, map[string]string{"message": "renewal successful", "output": output})
}

func (h *CertHandler) RequestCert(c echo.Context) error {
	var req struct {
		Domain  string `json:"domain"`
		Webroot string `json:"webroot"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if req.Domain == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "domain is required"})
	}

	output, err := h.certbot.RequestCert(req.Domain, req.Webroot)

	logEntry := model.OperationLog{
		Type:   "cert_request",
		Target: req.Domain,
		Status: "success",
	}
	if err != nil {
		logEntry.Status = "failed"
		logEntry.Message = err.Error() + "\n" + output
		database.DB.Create(&logEntry)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error(), "output": output})
	}

	logEntry.Message = output
	database.DB.Create(&logEntry)

	return c.JSON(http.StatusOK, map[string]string{"message": "certificate requested", "output": output})
}

func (h *CertHandler) Revoke(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	var cert model.Certificate
	if database.DB.First(&cert, id).Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "certificate not found"})
	}

	output, err := h.certbot.Revoke(cert.CertPath)

	logEntry := model.OperationLog{
		Type:   "cert_revoke",
		Target: cert.Domain,
		Status: "success",
	}
	if err != nil {
		logEntry.Status = "failed"
		logEntry.Message = err.Error() + "\n" + output
		database.DB.Create(&logEntry)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error(), "output": output})
	}

	logEntry.Message = output
	database.DB.Create(&logEntry)

	database.DB.Delete(&cert)

	return c.JSON(http.StatusOK, map[string]string{"message": "certificate revoked"})
}
