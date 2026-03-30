package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/configmodule"
	appconfig "github.com/sieryo/invoice-extractor/internal/config"
)

type ConfigModuleHandler struct {
	settings *appconfig.SettingsService
}

func NewConfigModuleHandler(settings *appconfig.SettingsService) *ConfigModuleHandler {
	return &ConfigModuleHandler{settings: settings}
}

func (h *ConfigModuleHandler) List(c *fiber.Ctx) error {
	items := configmodule.ListModules(h.settings.CurrentFeatures().EnableCashflowXLSX)
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"items": items,
		"count": len(items),
	}, "config module list retrieved")
}

func (h *ConfigModuleHandler) Get(c *fiber.Ctx) error {
	moduleKey := c.Params("moduleKey")
	item, ok := configmodule.FindModule(moduleKey, h.settings.CurrentFeatures().EnableCashflowXLSX)
	if !ok {
		return SendError(c, fiber.StatusNotFound, "config module not found")
	}
	return SendSuccess(c, fiber.StatusOK, item, "config module retrieved")
}
