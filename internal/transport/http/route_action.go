package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerActionRoutes(protected fiber.Router) {
	actionHandler := handler.NewActionHandler(s.appCtx.ActionService, s.appCtx.FileStore)

	protected.Post("/collection/:id/actions", actionHandler.RunAction)
	protected.Post("/collection/:id/actions/artifacts", actionHandler.UploadActionArtifact)
	protected.Get("/collection/:id/action-specs", actionHandler.GetActionSpec)
	protected.Post("/collection/:id/action-specs/resolve", actionHandler.ResolveActionSpec)
	protected.Get("/collection/:id/actions", actionHandler.ListActions)
	protected.Get("/collection/:id/actions/:actionId", actionHandler.GetActionDetail)
	protected.Get("/collection/:id/actions/:actionId/outputs/:outputId/download", actionHandler.DownloadActionOutput)

	protected.Post("/collections/:id/actions", actionHandler.RunAction)
	protected.Post("/collections/:id/actions/artifacts", actionHandler.UploadActionArtifact)
	protected.Get("/collections/:id/action-specs", actionHandler.GetActionSpec)
	protected.Post("/collections/:id/action-specs/resolve", actionHandler.ResolveActionSpec)
	protected.Get("/collections/:id/actions", actionHandler.ListActions)
	protected.Get("/collections/:id/actions/:actionId", actionHandler.GetActionDetail)
	protected.Get("/collections/:id/actions/:actionId/outputs/:outputId/download", actionHandler.DownloadActionOutput)
}
