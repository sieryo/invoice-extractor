package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/auth"
)

type AuthHandler struct {
	authService *auth.AuthService
}

func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type RegisterRequest struct {
	Name       string `json:"name"`
	Alias      string `json:"alias"`
	CutoffDate int    `json:"cutoff_date"`
	NPWP       string `json:"npwp"`
	TKUID      string `json:"tku_id"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginByProfileIDRequest struct {
	ProfileID string `json:"profile_id"`
}

type ProfileInfoResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Alias      string    `json:"alias"`
	CutoffDate int       `json:"cutoff_date"`
	NPWP       string    `json:"npwp,omitempty"`
	TKUID      string    `json:"tku_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.Alias == "" {
		return SendError(c, fiber.StatusBadRequest, "name and alias are required")
	}
	if req.CutoffDate <= 0 || req.CutoffDate > 31 {
		return SendError(c, fiber.StatusBadRequest, "cutoff_date must be between 1 and 31")
	}

	if err := h.authService.Register(auth.RegisterProfileInput{
		Name:       req.Name,
		Alias:      req.Alias,
		CutoffDate: req.CutoffDate,
		NPWP:       req.NPWP,
		TKUID:      req.TKUID,
	}); err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusCreated, nil, "profile registered successfully")
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.Password == "" {
		return SendError(c, fiber.StatusBadRequest, "name and password are required")
	}

	sessionID, err := h.authService.Login(req.Name, req.Password)
	if err != nil {
		return SendError(c, fiber.StatusUnauthorized, "invalid credentials")
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"session_id": sessionID,
	}, "login successful")
}

func (h *AuthHandler) LoginByProfileID(c *fiber.Ctx) error {
	var req LoginByProfileIDRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.ProfileID == "" {
		return SendError(c, fiber.StatusBadRequest, "profile_id is required")
	}

	sessionID, err := h.authService.LoginByProfileID(req.ProfileID)
	if err != nil {
		return SendError(c, fiber.StatusUnauthorized, "invalid credentials")
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"session_id": sessionID,
	}, "login successful")
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionID := c.Locals("sessionID")
	if sessionID == nil {
		return SendError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	if err := h.authService.Logout(sessionID.(string)); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to logout")
	}

	return SendSuccess(c, fiber.StatusOK, nil, "logout successful")
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	sessionID := c.Locals("sessionID")
	if sessionID == nil {
		return SendError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	p, err := h.authService.GetProfileBySessionID(sessionID.(string))
	if err != nil {
		return SendError(c, fiber.StatusUnauthorized, "profile not found")
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"id":          p.ID,
		"name":        p.Name,
		"alias":       p.Alias,
		"cutoff_date": p.CutoffDate,
		"npwp":        p.NPWP,
		"tku_id":      p.TKUID,
		"created_at":  p.CreatedAt,
	}, "profile retrieved successfully")
}

func (h *AuthHandler) ListProfiles(c *fiber.Ctx) error {
	profiles, err := h.authService.ListProfiles()
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to retrieve profiles")
	}

	if len(profiles) == 0 {
		return SendSuccess(c, fiber.StatusOK, []ProfileInfoResponse{}, "profiles retrieved successfully")
	}

	resp := make([]ProfileInfoResponse, 0, len(profiles))
	for _, p := range profiles {
		resp = append(resp, ProfileInfoResponse{
			ID:         p.ID,
			Name:       p.Name,
			Alias:      p.Alias,
			CutoffDate: p.CutoffDate,
			NPWP:       p.NPWP,
			TKUID:      p.TKUID,
			CreatedAt:  p.CreatedAt,
		})
	}

	return SendSuccess(c, fiber.StatusOK, resp, "profiles retrieved successfully")
}
