package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerInvoiceRoutes(protected fiber.Router) {

	invoiceHandler := handler.NewInvoiceHandler(
		s.appCtx.InvoiceService,
		s.appCtx.FileStore,
		s.appCtx.JobService,
	)
	protected.Post("/invoice/export", invoiceHandler.ExportInvoices)
	protected.Post("/invoice/load", invoiceHandler.LoadInvoice)
	protected.Get("/invoice/list/:job_id", invoiceHandler.ListInvoices)

	protected.Get("/invoice/tax/:job_id/download", invoiceHandler.DownloadTaxInvoices)

}
