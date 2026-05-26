package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"nginx-certs-manager/internal/database"
	"nginx-certs-manager/internal/model"
)

func GetSettings(c echo.Context) error {
	var settings []model.Setting
	database.DB.Find(&settings)

	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	return c.JSON(http.StatusOK, result)
}

func UpdateSettings(c echo.Context) error {
	var req map[string]string
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	for key, value := range req {
		setting := model.Setting{Key: key, Value: value}
		database.DB.Save(&setting)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "settings updated"})
}

func GetLogs(c echo.Context) error {
	var logs []model.OperationLog
	database.DB.Order("created_at DESC").Limit(100).Find(&logs)
	return c.JSON(http.StatusOK, logs)
}
