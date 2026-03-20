package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerBuyerRoutes(protected fiber.Router) {
	buyerRegistryHandler := handler.NewBuyerRegistryHandler(
		s.appCtx.BuyerRegistryService,
	)

	protected.Get("/buyer/isloaded", buyerRegistryHandler.IsLoaded)
	protected.Post("/buyer/upload", buyerRegistryHandler.Update)

	config := protected.Group("/config/master-data")
	config.Get("/buyer", buyerRegistryHandler.List)
	config.Get("/buyer/spec", buyerRegistryHandler.Spec)
	config.Get("/buyer/status", buyerRegistryHandler.IsLoaded)
	config.Post("/buyer/upload", buyerRegistryHandler.Update)
}
