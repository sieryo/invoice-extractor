package http

import (
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerRoutes() {
	api := s.app.Group("/api")
	invoiceExtractHandler := handler.NewInvoiceExtractHandler(s.appCtx.JobService, s.appCtx.FileStore)
	api.Post("/extract_invoice", invoiceExtractHandler.Handle)
}
