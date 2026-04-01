package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerCategoryAccountRoutes(protected fiber.Router) {
	if s.appCtx.CashflowBillCategoryService == nil {
		return
	}

	categoryHandler := handler.NewCategoryAccountHandler(s.appCtx.CashflowBillCategoryService)

	config := protected.Group("/config/master-data/category-accounts")
	config.Get("/", categoryHandler.List)
	config.Get("/spec", categoryHandler.Spec)
	config.Get("/status", categoryHandler.Status)
	config.Post("/upload", categoryHandler.Update)
}
