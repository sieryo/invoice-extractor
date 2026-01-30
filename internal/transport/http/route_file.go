package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerFileRoutes(protected fiber.Router) {
	fileHandler := handler.NewFileHandler(s.appCtx.FileService)

	protected.Post("/file/upload", fileHandler.Upload)
	protected.Get("/collection/:collectionId/files", fileHandler.ListByCollection)
}
