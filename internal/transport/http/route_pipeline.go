package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerPipelineRoutes(protected fiber.Router) {
	pipelineHandler := handler.NewCollectionPipelineHandler(s.appCtx.IngestService)

	protected.Get("/collection/:id/documents", pipelineHandler.ListDocuments)
	protected.Get("/collection/:id/history", pipelineHandler.ListHistory)
	protected.Get("/collection/:id/history/:historyId/items", pipelineHandler.ListHistoryItems)

	protected.Get("/collections/:id/documents", pipelineHandler.ListDocuments)
	protected.Get("/collections/:id/history", pipelineHandler.ListHistory)
	protected.Get("/collections/:id/history/:historyId/items", pipelineHandler.ListHistoryItems)
}
