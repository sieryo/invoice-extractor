package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/sieryo/invoice-extractor/internal/app/buyer"
)

type BuyerRegistryHandler struct {
	service *buyer.BuyerRegistryService
}

func NewBuyerRegistryHandler(service *buyer.BuyerRegistryService) *BuyerRegistryHandler {
	return &BuyerRegistryHandler{
		service: service,
	}
}

func (h *BuyerRegistryHandler) List(c *fiber.Ctx) error {
	buyers := h.service.List()
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"buyers": buyers,
		"count":  len(buyers),
	}, "buyer list retrieved")
}

func (h *BuyerRegistryHandler) Update(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "file required")
	}

	tmpPath := h.service.TempFilePath()
	if err := c.SaveFile(file, tmpPath); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to save file")
	}

	count, err := h.service.Update(tmpPath)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"count": count,
	}, "buyer data uploaded successfully")
}

func (h *BuyerRegistryHandler) IsLoaded(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"loaded": h.service.IsLoaded(),
	}, "buyer data status retrieved")
}
