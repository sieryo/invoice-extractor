package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerPipelineRoutes(protected fiber.Router) {
	pipelineHandler := handler.NewCollectionPipelineHandler(
		s.appCtx.IngestService,
		s.appCtx.SpreadsheetService,
		s.appCtx.InvoiceService,
	)

	protected.Get("/collection/:id/documents", pipelineHandler.ListDocuments)
	protected.Get("/collection/:id/history", pipelineHandler.ListHistory)
	protected.Get("/collection/:id/history/:historyId/items", pipelineHandler.ListHistoryItems)
	protected.Get("/collection/:id/documents/:documentId/invoice", pipelineHandler.GetDocumentInvoice)
	protected.Get("/collection/:id/documents/:documentId/spreadsheet/meta", pipelineHandler.GetSpreadsheetWorkbookMeta)
	protected.Get("/collection/:id/documents/:documentId/spreadsheet/rows/stream", pipelineHandler.StreamSpreadsheetSheetRows)
	// Backward-compatible aliases for existing FE path.
	protected.Get("/collection/:id/documents/:documentId/cashflow/meta", pipelineHandler.GetSpreadsheetWorkbookMeta)
	protected.Get("/collection/:id/documents/:documentId/cashflow/rows/stream", pipelineHandler.StreamSpreadsheetSheetRows)

	protected.Get("/collections/:id/documents", pipelineHandler.ListDocuments)
	protected.Get("/collections/:id/history", pipelineHandler.ListHistory)
	protected.Get("/collections/:id/history/:historyId/items", pipelineHandler.ListHistoryItems)
	protected.Get("/collections/:id/documents/:documentId/invoice", pipelineHandler.GetDocumentInvoice)
	protected.Get("/collections/:id/documents/:documentId/spreadsheet/meta", pipelineHandler.GetSpreadsheetWorkbookMeta)
	protected.Get("/collections/:id/documents/:documentId/spreadsheet/rows/stream", pipelineHandler.StreamSpreadsheetSheetRows)
	// Backward-compatible aliases for existing FE path.
	protected.Get("/collections/:id/documents/:documentId/cashflow/meta", pipelineHandler.GetSpreadsheetWorkbookMeta)
	protected.Get("/collections/:id/documents/:documentId/cashflow/rows/stream", pipelineHandler.StreamSpreadsheetSheetRows)
}
