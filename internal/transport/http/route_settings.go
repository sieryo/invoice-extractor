package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerSettingsRoutes(protected fiber.Router) {
	h := handler.NewSettingsHandler(s.appCtx.AuthService, s.appCtx.ModuleActivationService)

	settings := protected.Group("/settings")
	settings.Get("/profile", h.GetProfile)
	settings.Put("/profile", h.UpdateProfile)
	settings.Get("/modules", h.GetModules)
	settings.Put("/modules", h.UpdateModules)
}
