package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerBukpotRequestConfigRoutes(protected fiber.Router) {
	if s.appCtx.BukpotRequestConfigService == nil {
		return
	}

	configHandler := handler.NewBukpotRequestConfigHandler(
		s.appCtx.BukpotRequestConfigService,
	)

	config := protected.Group("/config/profile")
	config.Get("/bukpot-request", configHandler.Get)
	config.Get("/bukpot-request/spec", configHandler.Spec)
	config.Get("/bukpot-request/status", configHandler.Status)
	config.Put("/bukpot-request", configHandler.Update)
}
