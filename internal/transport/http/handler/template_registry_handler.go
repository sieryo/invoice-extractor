package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
)

type TemplateRegistryHandler struct {
	service *template.TemplateRegistryService
}

func NewTemplateRegistryHandler(service *template.TemplateRegistryService) *TemplateRegistryHandler {
	return &TemplateRegistryHandler{
		service: service,
	}
}

func (h *TemplateRegistryHandler) List(c *fiber.Ctx) error {
	templates := h.service.List()
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"templates": templates,
		"count":     len(templates),
	}, "template list retrieved")
}
