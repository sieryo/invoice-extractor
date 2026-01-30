package handler

import (
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
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return SendError(c, fiber.StatusBadRequest, "username and password are required")
	}

	if err := h.authService.Register(req.Username, req.Password); err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusCreated, nil, "user registered successfully")
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" || req.Password == "" {
		return SendError(c, fiber.StatusBadRequest, "username and password are required")
	}

	sessionID, err := h.authService.Login(req.Username, req.Password)
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

	u, err := h.authService.GetUserBySessionID(sessionID.(string))
	if err != nil {
		return SendError(c, fiber.StatusUnauthorized, "user not found")
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"id":         u.ID,
		"username":   u.Username,
		"created_at": u.CreatedAt,
	}, "user retrieved successfully")
}
