package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/auth"
	"github.com/sieryo/invoice-extractor/internal/app/moduleactivation"
	"github.com/sieryo/invoice-extractor/internal/app/session"
)

type SettingsHandler struct {
	authService   *auth.AuthService
	moduleService *moduleactivation.Service
}

func NewSettingsHandler(authService *auth.AuthService, moduleService *moduleactivation.Service) *SettingsHandler {
	return &SettingsHandler{
		authService:   authService,
		moduleService: moduleService,
	}
}

type UpdateProfileSettingsRequest struct {
	Name       string `json:"name"`
	Alias      string `json:"alias"`
	CutoffDate int    `json:"cutoffDate"`
	NPWP       string `json:"npwp"`
	TKUID      string `json:"tkuId"`
}

type UpdateModuleSettingsRequest struct {
	Modules map[string]bool `json:"modules"`
}

func (h *SettingsHandler) GetProfile(c *fiber.Ctx) error {
	sessionID, _ := c.Locals("sessionID").(string)
	if sessionID == "" {
		return fiber.ErrUnauthorized
	}

	profile, err := h.authService.GetProfileBySessionID(sessionID)
	if err != nil {
		return SendError(c, fiber.StatusUnauthorized, "profile not found")
	}

	currentSession, _ := h.authService.GetSession(sessionID)
	latestSession, _ := h.authService.GetLatestSessionByProfileID(profile.ID)

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"profile": fiber.Map{
			"id":         profile.ID,
			"name":       profile.Name,
			"alias":      profile.Alias,
			"cutoffDate": profile.CutoffDate,
			"npwp":       profile.NPWP,
			"tkuId":      profile.TKUID,
			"createdAt":  profile.CreatedAt,
		},
		"session": fiber.Map{
			"currentLoginAt":  timeOrNil(currentSession),
			"lastLoginAt":     timeOrNil(latestSession),
			"sessionExpiresAt": expiresAtOrNil(currentSession),
		},
	}, "settings profile retrieved")
}

func (h *SettingsHandler) UpdateProfile(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	if profileID == "" {
		return fiber.ErrUnauthorized
	}

	var payload UpdateProfileSettingsRequest
	if err := c.BodyParser(&payload); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	profile, err := h.authService.UpdateProfile(profileID, auth.UpdateProfileInput{
		Name:       payload.Name,
		Alias:      payload.Alias,
		CutoffDate: payload.CutoffDate,
		NPWP:       payload.NPWP,
		TKUID:      payload.TKUID,
	})
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"id":         profile.ID,
		"name":       profile.Name,
		"alias":      profile.Alias,
		"cutoffDate": profile.CutoffDate,
		"npwp":       profile.NPWP,
		"tkuId":      profile.TKUID,
		"createdAt":  profile.CreatedAt,
	}, "settings profile updated")
}

func (h *SettingsHandler) GetModules(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	if profileID == "" {
		return fiber.ErrUnauthorized
	}

	settings, err := h.moduleService.Load(profileID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat aktivasi modul")
	}
	items, err := h.moduleService.Preferences(profileID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat aktivasi modul")
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"schemaVersion": settings.SchemaVersion,
		"modules":       settings.Modules,
		"items":         items,
	}, "settings modules retrieved")
}

func (h *SettingsHandler) UpdateModules(c *fiber.Ctx) error {
	profileID, _ := c.Locals("profileId").(string)
	if profileID == "" {
		return fiber.ErrUnauthorized
	}

	var payload UpdateModuleSettingsRequest
	if err := c.BodyParser(&payload); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	settings, err := h.moduleService.Update(profileID, moduleactivation.ModuleSettings{
		Modules: payload.Modules,
	})
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}
	items, err := h.moduleService.Preferences(profileID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "gagal memuat aktivasi modul")
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"schemaVersion": settings.SchemaVersion,
		"modules":       settings.Modules,
		"items":         items,
	}, "settings modules updated")
}

func timeOrNil(value any) any {
	typed, ok := value.(*session.Session)
	if !ok || typed == nil || typed.CreatedAt.IsZero() {
		return nil
	}
	return typed.CreatedAt
}

func expiresAtOrNil(value any) any {
	typed, ok := value.(*session.Session)
	if !ok || typed == nil || typed.ExpiresAt.IsZero() {
		return nil
	}
	return typed.ExpiresAt
}
