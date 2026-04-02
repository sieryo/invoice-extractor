package handler

import (
	"errors"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/moduleactivation"
	"github.com/sieryo/invoice-extractor/internal/app/configmodule"
	appconfig "github.com/sieryo/invoice-extractor/internal/config"
)

type ConfigModuleHandler struct {
	settings *appconfig.SettingsService
	modules  *moduleactivation.Service
	pages    *configmodule.PageService
}

func NewConfigModuleHandler(
	settings *appconfig.SettingsService,
	modules *moduleactivation.Service,
	pages *configmodule.PageService,
) *ConfigModuleHandler {
	return &ConfigModuleHandler{
		settings: settings,
		modules:  modules,
		pages:    pages,
	}
}

func (h *ConfigModuleHandler) List(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	items := configmodule.ListModules(h.settings.CurrentFeatures().EnableCashflowXLSX)
	filtered := make([]configmodule.ModuleSpec, 0, len(items))
	for _, item := range items {
		if item.Key == "app_modules" {
			continue
		}
		enabled, _, err := h.modules.IsEnabledForConfigGroup(profileID, item.Group)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "gagal memuat daftar konfigurasi")
		}
		if !enabled {
			continue
		}
		filtered = append(filtered, item)
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"items": filtered,
		"count": len(filtered),
	}, "config module list retrieved")
}

func (h *ConfigModuleHandler) Get(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	moduleKey := c.Params("moduleKey")
	item, ok := configmodule.FindModule(moduleKey, h.settings.CurrentFeatures().EnableCashflowXLSX)
	if !ok {
		return SendError(c, fiber.StatusNotFound, "config module not found")
	}
	if item.Key == "app_modules" {
		return SendError(c, fiber.StatusNotFound, "config module not found")
	}
	enabled, _, err := h.modules.IsEnabledForConfigGroup(profileID, item.Group)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat modul konfigurasi")
	}
	if !enabled {
		return SendError(c, fiber.StatusNotFound, "config module not found")
	}
	return SendSuccess(c, fiber.StatusOK, item, "config module retrieved")
}

func (h *ConfigModuleHandler) Page(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	moduleKey := c.Params("moduleKey")
	item, ok := configmodule.FindModule(moduleKey, h.settings.CurrentFeatures().EnableCashflowXLSX)
	if !ok || item.Key == "app_modules" {
		return SendError(c, fiber.StatusNotFound, "config module not found")
	}
	enabled, _, err := h.modules.IsEnabledForConfigGroup(profileID, item.Group)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat halaman konfigurasi")
	}
	if !enabled {
		return SendError(c, fiber.StatusNotFound, "config module not found")
	}
	page, err := h.pages.Page(profileID, moduleKey, h.settings.CurrentFeatures().EnableCashflowXLSX)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SendError(c, fiber.StatusNotFound, "config module not found")
		}
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat halaman konfigurasi")
	}
	return SendSuccess(c, fiber.StatusOK, page, "config module page retrieved")
}

func (h *ConfigModuleHandler) UpdateBlock(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	moduleKey := c.Params("moduleKey")
	blockKey := c.Params("blockKey")
	item, ok := configmodule.FindModule(moduleKey, h.settings.CurrentFeatures().EnableCashflowXLSX)
	if !ok || item.Key == "app_modules" {
		return SendError(c, fiber.StatusNotFound, "config block not found")
	}
	enabled, _, err := h.modules.IsEnabledForConfigGroup(profileID, item.Group)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal menyimpan konfigurasi")
	}
	if !enabled {
		return SendError(c, fiber.StatusNotFound, "config block not found")
	}

	var payload configmodule.FormBlockInput
	if err := c.BodyParser(&payload); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.pages.UpdateFormBlock(profileID, moduleKey, blockKey, payload); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SendError(c, fiber.StatusNotFound, "config block not found")
		}
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{"updated": true}, "config block updated successfully")
}

func (h *ConfigModuleHandler) UploadBlock(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	moduleKey := c.Params("moduleKey")
	blockKey := c.Params("blockKey")
	item, ok := configmodule.FindModule(moduleKey, h.settings.CurrentFeatures().EnableCashflowXLSX)
	if !ok || item.Key == "app_modules" {
		return SendError(c, fiber.StatusNotFound, "config block not found")
	}
	enabled, _, err := h.modules.IsEnabledForConfigGroup(profileID, item.Group)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal upload konfigurasi")
	}
	if !enabled {
		return SendError(c, fiber.StatusNotFound, "config block not found")
	}

	file, err := c.FormFile("file")
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "file required")
	}
	if file.Size <= 0 {
		return SendError(c, fiber.StatusBadRequest, "file kosong atau tidak valid")
	}

	tmpFile, err := os.CreateTemp("", "config-upload-*")
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to prepare upload")
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := c.SaveFile(file, tmpPath); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to save file")
	}

	count, issues, err := h.pages.UploadRegistryBlock(profileID, moduleKey, blockKey, file.Filename, file.Size, tmpPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SendError(c, fiber.StatusNotFound, "config block not found")
		}
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"count":  count,
		"issues": issues,
	}, "config block uploaded successfully")
}
