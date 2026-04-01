package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
)

type BukpotActionProfileHandler struct {
	service *appbukpot.ActionProfileService
	key     appbukpot.ActionProfileKey
}

func NewBukpotActionProfileHandler(
	service *appbukpot.ActionProfileService,
	key appbukpot.ActionProfileKey,
) *BukpotActionProfileHandler {
	return &BukpotActionProfileHandler{service: service, key: key}
}

func (h *BukpotActionProfileHandler) Get(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	cfg, err := h.service.Load(profileID, h.key)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat default profil bukpot")
	}
	return SendSuccess(c, fiber.StatusOK, cfg, "bukpot action profile retrieved")
}

func (h *BukpotActionProfileHandler) Spec(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Spec(h.key), "bukpot action profile spec retrieved")
}

func (h *BukpotActionProfileHandler) Status(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	return SendSuccess(c, fiber.StatusOK, h.service.Status(profileID, h.key), "bukpot action profile status retrieved")
}

func (h *BukpotActionProfileHandler) Update(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	var payload appbukpot.ActionProfile
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
	}, "bukpot action profile updated successfully")
}
