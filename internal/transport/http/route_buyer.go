package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerBuyerRoutes(protected fiber.Router) {
	buyerHandler := handler.NewBuyerUploadHandler(
		s.appCtx.BuyerRegistry,
		s.appCtx.BuyerStore,
		s.appCtx.RootDir,
	)

	protected.Get("/buyer/isloaded", buyerHandler.IsLoaded)
	protected.Post("/buyer/upload", buyerHandler.Handle)
}
