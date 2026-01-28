package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerAuthRoutes(api fiber.Router) {
	authHandler := handler.NewAuthHandler(s.appCtx.AuthService)

	// Public auth routes
	api.Post("/auth/register", authHandler.Register)
	api.Post("/auth/login", authHandler.Login)

	// Protected auth routes
	protected := api.Group("", s.AuthMiddleware())
	protected.Post("/auth/logout", authHandler.Logout)
}
