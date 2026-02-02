package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerCollectionRoutes(protected fiber.Router) {
	collectionHandler := handler.NewCollectionHandler(s.appCtx.CollectionService)
	fileHandler := handler.NewFileHandler(s.appCtx.FileService)

	protected.Post("/collection/create", collectionHandler.CreateCollection)
	protected.Get("/collection/list", collectionHandler.ListUserCollections)

	protected.Get("/collection/:id", collectionHandler.GetCollectionByID)
	protected.Delete("/collection/:id", collectionHandler.DeleteCollection)
	protected.Get("/collection/:id/files", fileHandler.ListByCollection)
	protected.Post("/collection/:id/files", fileHandler.Upload)

	protected.Get("/file/:id", fileHandler.GetFileObjectByID)
	protected.Delete("/file/:id", fileHandler.DeleteFile)
	protected.Post("/file/delete", fileHandler.DeleteFilesBulk)
}
