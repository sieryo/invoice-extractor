package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerAuthRoutes(api fiber.Router) {
	authHandler := handler.NewAuthHandler(s.appCtx.AuthService)

	api.Post("/auth/register", authHandler.Register)
	api.Post("/auth/login", authHandler.Login)
	api.Post("/auth/login/by-profile-id", authHandler.LoginByProfileID)
	api.Post("/auth/login/by-id", authHandler.LoginByProfileID)
	api.Get("/auth/profiles", authHandler.ListProfiles)
	api.Get("/auth/users", authHandler.ListProfiles)

	protected := api.Group("", s.AuthMiddleware())
	protected.Get("/auth/me", authHandler.Me)
	protected.Post("/auth/logout", authHandler.Logout)
}
