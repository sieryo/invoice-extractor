package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	appconfig "github.com/sieryo/invoice-extractor/internal/config"
)

type AppSettingsHandler struct {
	service *appconfig.SettingsService
}

func NewAppSettingsHandler(service *appconfig.SettingsService) *AppSettingsHandler {
	return &AppSettingsHandler{service: service}
}

func (h *AppSettingsHandler) Get(c *fiber.Ctx) error {
	settings, err := h.service.Load()
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat pengaturan modul aplikasi")
	}
	return SendSuccess(c, fiber.StatusOK, settings, "app settings retrieved")
}

func (h *AppSettingsHandler) Status(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Status(), "app settings status retrieved")
}

func (h *AppSettingsHandler) Update(c *fiber.Ctx) error {
	var payload appconfig.AppSettings
	if err := c.BodyParser(&payload); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}
	settings, err := h.service.Update(payload)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}
	document.SetFeatureFlags(document.FeatureFlags{
		EnableCashflowXLSX: settings.Features.EnableCashflowXLSX,
	})
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"settings": settings,
		"status":   h.service.Status(),
	}, "app settings updated successfully")
}
