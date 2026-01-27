package http

import (
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerRoutes() {
	api := s.app.Group("/api")

	authHandler := handler.NewAuthHandler(s.appCtx.AuthService)
	api.Post("/auth/register", authHandler.Register)
	api.Post("/auth/login", authHandler.Login)

	protected := api.Group("", s.AuthMiddleware())

	protected.Post("/auth/logout", authHandler.Logout)

	invoiceExtractHandler := handler.NewInvoiceExtractHandler(s.appCtx.JobService, s.appCtx.FileStore)
	protected.Post("/extract_invoice", invoiceExtractHandler.Handle)

	jobHandler := handler.NewJobHandler(s.appCtx.JobService)
	protected.Get("/job/list", jobHandler.ListJobs)
	protected.Get("/job/:id", jobHandler.GetJobByID)
}
