package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerPipelineRoutes(protected fiber.Router) {
	pipelineHandler := handler.NewCollectionPipelineHandler(
		s.appCtx.IngestService,
		s.appCtx.InvoiceService,
	)

	protected.Get("/collection/:id/documents", pipelineHandler.ListDocuments)
	protected.Post("/collection/:id/documents/:documentId/replace-source", pipelineHandler.ReplaceDocumentSource)
	protected.Get("/collection/:id/history", pipelineHandler.ListHistory)
	protected.Get("/collection/:id/history/:historyId/items", pipelineHandler.ListHistoryItems)
	protected.Get("/collection/:id/documents/:documentId/invoice", pipelineHandler.GetDocumentInvoice)

	protected.Get("/collections/:id/documents", pipelineHandler.ListDocuments)
	protected.Post("/collections/:id/documents/:documentId/replace-source", pipelineHandler.ReplaceDocumentSource)
	protected.Get("/collections/:id/history", pipelineHandler.ListHistory)
	protected.Get("/collections/:id/history/:historyId/items", pipelineHandler.ListHistoryItems)
	protected.Get("/collections/:id/documents/:documentId/invoice", pipelineHandler.GetDocumentInvoice)
}
