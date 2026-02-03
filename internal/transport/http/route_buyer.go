package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerBuyerRoutes(protected fiber.Router) {
	// Legacy buyer upload handler (for backward compatibility)
	buyerHandler := handler.NewBuyerUploadHandler(
		s.appCtx.BuyerRegistry,
		s.appCtx.BuyerStore,
		s.appCtx.RootDir,
	)

	protected.Get("/buyer/isloaded", buyerHandler.IsLoaded)
	protected.Post("/buyer/upload", buyerHandler.Handle)

	// New buyer registry routes at /config/master-data
	buyerRegistryHandler := handler.NewBuyerRegistryHandler(
		s.appCtx.BuyerRegistryService,
	)

	config := protected.Group("/config/master-data")
	config.Get("/buyer", buyerRegistryHandler.List)
	config.Post("/buyer/upload", buyerRegistryHandler.Update)
	config.Get("/buyer/status", buyerRegistryHandler.IsLoaded)
}
