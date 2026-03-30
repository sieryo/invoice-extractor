package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/buildinfo"
	"github.com/sieryo/invoice-extractor/internal/config"
)

type MetaHandler struct {
	settings *config.SettingsService
}

func NewMetaHandler(settings *config.SettingsService) *MetaHandler {
	return &MetaHandler{settings: settings}
}

type AppMetaResponse struct {
	Version     string `json:"version"`
	Environment string `json:"environment"`
	BaseAPIPath string `json:"base_api_path"`
	Features    struct {
		CashflowXLSXEnabled bool `json:"cashflowXlsxEnabled"`
	} `json:"features"`
}

func (h *MetaHandler) GetMeta(c *fiber.Ctx) error {
	resp := AppMetaResponse{
		Version:     buildinfo.Version,
		Environment: buildinfo.Environment,
		BaseAPIPath: buildinfo.BaseAPIPath,
	}
	resp.Features.CashflowXLSXEnabled = h.settings.CurrentFeatures().EnableCashflowXLSX

	return SendSuccess(c, fiber.StatusOK, AppMetaResponse{
		Version:     resp.Version,
		Environment: resp.Environment,
		BaseAPIPath: resp.BaseAPIPath,
		Features:    resp.Features,
	}, "meta retrieved successfully")
}
