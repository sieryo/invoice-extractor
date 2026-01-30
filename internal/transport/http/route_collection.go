package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerCollectionRoutes(protected fiber.Router) {
	collectionHandler := handler.NewCollectionHandler(s.appCtx.CollectionService)
	fileHandler := handler.NewFileHandler(s.appCtx.FileService)

	protected.Post("/collection/upload", fileHandler.Upload)
	protected.Post("/collection/create", collectionHandler.CreateCollection)
	protected.Get("/collection/:id", collectionHandler.GetCollectionByID)
	protected.Get("/collection/list", collectionHandler.ListUserCollections)
	protected.Get("/collection/:id/files", fileHandler.ListByCollection)
}
