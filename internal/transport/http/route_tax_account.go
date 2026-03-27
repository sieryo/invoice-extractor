package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerTaxAccountRoutes(protected fiber.Router) {
	if !s.appCtx.Features.EnableCashflowXLSX || s.appCtx.TaxAccountService == nil {
		return
	}

	taxAccountHandler := handler.NewTaxAccountHandler(
		s.appCtx.TaxAccountService,
	)

	config := protected.Group("/config/master-data")
	config.Get("/tax-accounts", taxAccountHandler.List)
	config.Get("/tax-accounts/spec", taxAccountHandler.Spec)
	config.Get("/tax-accounts/status", taxAccountHandler.Status)
	config.Post("/tax-accounts/upload", taxAccountHandler.Update)
}
