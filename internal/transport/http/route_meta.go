package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerMetaRoutes(api fiber.Router) {
	metaHandler := handler.NewMetaHandler(s.appCtx.Features)
	api.Get("/meta", metaHandler.GetMeta)
}
