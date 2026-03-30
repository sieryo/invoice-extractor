package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerConfigModuleRoutes(protected fiber.Router) {
	config := protected.Group("/config")
	h := handler.NewConfigModuleHandler(s.appCtx.Features)
	config.Get("/modules", h.List)
	config.Get("/modules/:moduleKey", h.Get)
}
