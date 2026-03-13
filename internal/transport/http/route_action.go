package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerActionRoutes(protected fiber.Router) {
	actionHandler := handler.NewActionHandler(s.appCtx.ActionService)

	protected.Post("/collection/:id/actions", actionHandler.RunAction)
	protected.Get("/collection/:id/actions", actionHandler.ListActions)
	protected.Get("/collection/:id/actions/:actionId", actionHandler.GetActionDetail)

	protected.Post("/collections/:id/actions", actionHandler.RunAction)
	protected.Get("/collections/:id/actions", actionHandler.ListActions)
	protected.Get("/collections/:id/actions/:actionId", actionHandler.GetActionDetail)
}
