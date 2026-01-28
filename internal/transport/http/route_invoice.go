package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerInvoiceRoutes(protected fiber.Router) {
	invoiceExtractHandler := handler.NewInvoiceExtractHandler(
		s.appCtx.JobService,
		s.appCtx.FileStore,
	)
	protected.Post("/extract_invoice", invoiceExtractHandler.Handle)

	invoiceHandler := handler.NewInvoiceHandler(
		s.appCtx.InvoiceService,
		s.appCtx.FileStore,
		s.appCtx.JobService,
	)
	protected.Post("/invoice/export", invoiceHandler.ExportInvoices)
}
