package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	appcashflowbill "github.com/sieryo/invoice-extractor/internal/app/cashflowbill"
)

type CashflowBillProfileConfigHandler struct {
	service *appcashflowbill.ProfileConfigService
	key     appcashflowbill.ProfileConfigKey
}

func NewCashflowBillProfileConfigHandler(
	service *appcashflowbill.ProfileConfigService,
	key appcashflowbill.ProfileConfigKey,
) *CashflowBillProfileConfigHandler {
	return &CashflowBillProfileConfigHandler{service: service, key: key}
}

func (h *CashflowBillProfileConfigHandler) Get(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	cfg, err := h.service.Load(profileID, h.key)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat default profil cashflow bills")
	}
	return SendSuccess(c, fiber.StatusOK, cfg, "cashflow bills profile config retrieved")
}

func (h *CashflowBillProfileConfigHandler) Spec(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Spec(h.key), "cashflow bills profile config spec retrieved")
}

func (h *CashflowBillProfileConfigHandler) Status(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	return SendSuccess(c, fiber.StatusOK, h.service.Status(profileID, h.key), "cashflow bills profile config status retrieved")
}

func (h *CashflowBillProfileConfigHandler) Update(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	var payload appcashflowbill.ProfileConfig
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
	}, "cashflow bills profile config updated successfully")
}
