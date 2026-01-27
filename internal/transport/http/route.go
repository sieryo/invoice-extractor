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

	buyerHandler := handler.NewBuyerUploadHandler(s.appCtx.BuyerRegistry, s.appCtx.BuyerStore, s.appCtx.RootDir)
	protected.Get("/buyer/isloaded", buyerHandler.IsLoaded)
	protected.Post("/buyer/upload", buyerHandler.Handle)

	invoiceHandler := handler.NewInvoiceHandler(s.appCtx.InvoiceService, s.appCtx.FileStore, s.appCtx.JobService)
	protected.Post("/invoice/export", invoiceHandler.ExportInvoices)
}
