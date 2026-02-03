package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerTemplateRoutes(protected fiber.Router) {
	templateHandler := handler.NewTemplateRegistryHandler(
		s.appCtx.TemplateRegistryService,
	)

	config := protected.Group("/config/invoice/template")
	config.Get("/", templateHandler.List)
}
