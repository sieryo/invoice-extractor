package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/buildinfo"
)

type MetaHandler struct{}

func NewMetaHandler() *MetaHandler {
	return &MetaHandler{}
}

type AppMetaResponse struct {
	Version     string `json:"version"`
	Environment string `json:"environment"`
	BaseAPIPath string `json:"base_api_path"`
}

func (h *MetaHandler) GetMeta(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, AppMetaResponse{
		Version:     buildinfo.Version,
		Environment: buildinfo.Environment,
		BaseAPIPath: buildinfo.BaseAPIPath,
	}, "meta retrieved successfully")
}
