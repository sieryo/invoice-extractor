package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerTaxInvoiceRoutes(protected fiber.Router) {

	taxInvoiceRenameHandler := handler.NewTaxInvoiceRenameHandler(
		s.appCtx.JobService,
		s.appCtx.FileStore,
	)
	protected.Post("/invoice/tax/rename", taxInvoiceRenameHandler.Handle)
}
