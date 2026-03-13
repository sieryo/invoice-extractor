package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerUploadRoutes(protected fiber.Router) {
	uploadHandler := handler.NewUploadSessionHandler(s.appCtx.IngestService)

	protected.Post("/collection/:id/upload/session", uploadHandler.StartSession)
	protected.Post("/upload/session/:id/chunk", uploadHandler.UploadChunk)
	protected.Post("/upload/session/:id/finalize", uploadHandler.FinalizeSession)
	protected.Get("/upload/session/:id", uploadHandler.GetSession)
}
