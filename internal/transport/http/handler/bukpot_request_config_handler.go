package handler

import (
	"github.com/gofiber/fiber/v2"
	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
)

type BukpotRequestConfigHandler struct {
	service *appbukpot.RequestConfigService
}

func NewBukpotRequestConfigHandler(service *appbukpot.RequestConfigService) *BukpotRequestConfigHandler {
	return &BukpotRequestConfigHandler{service: service}
}

func (h *BukpotRequestConfigHandler) Get(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	cfg, err := h.service.Load(profileID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat konfigurasi request bukpot")
	}
	return SendSuccess(c, fiber.StatusOK, cfg, "bukpot request config retrieved")
}

func (h *BukpotRequestConfigHandler) Spec(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Spec(), "bukpot request config spec retrieved")
}

func (h *BukpotRequestConfigHandler) Status(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	return SendSuccess(c, fiber.StatusOK, h.service.Status(profileID), "bukpot request config status retrieved")
}

func (h *BukpotRequestConfigHandler) Update(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	var payload appbukpot.RequestConfig
	if err := c.BodyParser(&payload); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}
	cfg, err := h.service.Update(profileID, payload)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"config": cfg,
		"status": h.service.Status(profileID),
	}, "bukpot request config updated successfully")
}
