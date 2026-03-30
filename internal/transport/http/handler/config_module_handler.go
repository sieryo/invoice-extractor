package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/configmodule"
	appconfig "github.com/sieryo/invoice-extractor/internal/config"
)

type ConfigModuleHandler struct {
	features appconfig.FeatureFlags
}

func NewConfigModuleHandler(features appconfig.FeatureFlags) *ConfigModuleHandler {
	return &ConfigModuleHandler{features: features}
}

func (h *ConfigModuleHandler) List(c *fiber.Ctx) error {
	items := configmodule.ListModules(h.features.EnableCashflowXLSX)
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"items": items,
		"count": len(items),
	}, "config module list retrieved")
}

func (h *ConfigModuleHandler) Get(c *fiber.Ctx) error {
	moduleKey := c.Params("moduleKey")
	item, ok := configmodule.FindModule(moduleKey, h.features.EnableCashflowXLSX)
	if !ok {
		return SendError(c, fiber.StatusNotFound, "config module not found")
	}
	return SendSuccess(c, fiber.StatusOK, item, "config module retrieved")
}
