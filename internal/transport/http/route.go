package http

func (s *Server) registerRoutes() {
	api := s.app.Group("/api")
	s.registerMetaRoutes(api)

	// Register authentication routes (public and protected)
	s.registerAuthRoutes(api)

	// Create protected route group for authenticated endpoints
	protected := api.Group("", s.AuthMiddleware())

	// Register domain-specific routes
	s.registerJobRoutes(protected)
	s.registerBuyerRoutes(protected)
	s.registerTaxAccountRoutes(protected)
	s.registerBukpotRequestConfigRoutes(protected)
	s.registerInvoiceRoutes(protected)
	s.registerCollectionRoutes(protected)
	s.registerTemplateRoutes(protected)
	s.registerUploadRoutes(protected)
	s.registerActionRoutes(protected)
	s.registerPipelineRoutes(protected)

	// Register fe route
	s.registerFrontendRoute(s.app)
}
