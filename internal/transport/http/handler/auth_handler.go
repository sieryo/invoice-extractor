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
	Username string `json:"username"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginByIDRequest struct {
	UserID string `json:"user_id"`
}

type UserInfoResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Username == "" {
		return SendError(c, fiber.StatusBadRequest, "username are required")
	}

	if err := h.authService.Register(req.Username); err != nil {
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

func (h *AuthHandler) LoginByID(c *fiber.Ctx) error {
	var req LoginByIDRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.UserID == "" {
		return SendError(c, fiber.StatusBadRequest, "user_id is required")
	}

	sessionID, err := h.authService.LoginByID(req.UserID)
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

func (h *AuthHandler) ListUsers(c *fiber.Ctx) error {
	users, err := h.authService.ListUsers()
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to retrieve users")
	}

	if len(users) == 0 {
		return SendSuccess(c, fiber.StatusOK, []UserInfoResponse{}, "users retrieved successfully")
	}

	resp := make([]UserInfoResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, UserInfoResponse{
			ID:        u.ID,
			Username:  u.Username,
			CreatedAt: u.CreatedAt,
		})
	}

	return SendSuccess(c, fiber.StatusOK, resp, "users retrieved successfully")
}
