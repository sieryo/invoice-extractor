package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerJobRoutes(protected fiber.Router) {
	jobHandler := handler.NewJobHandler(s.appCtx.JobService)

	protected.Get("/job/list", jobHandler.ListJobs)
	protected.Get("/job/:id", jobHandler.GetJobByID)
}
