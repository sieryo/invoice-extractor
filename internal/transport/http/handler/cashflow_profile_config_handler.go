package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
)

type CashflowProfileConfigHandler struct {
	service *appcashflow.ProfileConfigService
	key     appcashflow.ProfileConfigKey
}

func NewCashflowProfileConfigHandler(
	service *appcashflow.ProfileConfigService,
	key appcashflow.ProfileConfigKey,
) *CashflowProfileConfigHandler {
	return &CashflowProfileConfigHandler{service: service, key: key}
}

func (h *CashflowProfileConfigHandler) Get(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	cfg, err := h.service.Load(profileID, h.key)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat default profil cashflow")
	}
	return SendSuccess(c, fiber.StatusOK, cfg, "cashflow profile config retrieved")
}

func (h *CashflowProfileConfigHandler) Spec(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Spec(h.key), "cashflow profile config spec retrieved")
}

func (h *CashflowProfileConfigHandler) Status(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	return SendSuccess(c, fiber.StatusOK, h.service.Status(profileID, h.key), "cashflow profile config status retrieved")
}

func (h *CashflowProfileConfigHandler) Update(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	var payload appcashflow.ProfileConfig
	if err := c.BodyParser(&payload); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}
	payload.ConfigKey = strings.TrimSpace(string(h.key))
	cfg, err := h.service.Update(profileID, h.key, payload)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"config": cfg,
		"status": h.service.Status(profileID, h.key),
	}, "cashflow profile config updated successfully")
}
