package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerJobRoutes(protected fiber.Router) {
	jobHandler := handler.NewJobHandler(s.appCtx.JobService, s.appCtx.FileStore)

	protected.Post("/job/create", jobHandler.CreateJob)
	protected.Get("/job/list", jobHandler.ListJobs)
	protected.Get("/job/:id", jobHandler.GetJobByID)
	protected.Get("/job/:id/storage", jobHandler.GetJobStorage)
	protected.Post("/job/archive", jobHandler.ArchiveJob)
	protected.Post("/job/unarchive", jobHandler.UnarchiveJob)
	protected.Post("/job/start", jobHandler.StartJob)
	protected.Delete("/job/:id", jobHandler.DeleteJob)
}
