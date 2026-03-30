package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerAppSettingsRoutes(protected fiber.Router) {
	if s.appCtx.SettingsService == nil {
		return
	}

	h := handler.NewAppSettingsHandler(s.appCtx.SettingsService)
	config := protected.Group("/config/app")
	config.Get("/settings", h.Get)
	config.Get("/settings/status", h.Status)
	config.Put("/settings", h.Update)
}
