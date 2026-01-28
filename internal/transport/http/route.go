package http

func (s *Server) registerRoutes() {
	api := s.app.Group("/api")

	// Register authentication routes (public and protected)
	s.registerAuthRoutes(api)

	// Create protected route group for authenticated endpoints
	protected := api.Group("", s.AuthMiddleware())

	// Register domain-specific routes
	s.registerJobRoutes(protected)
	s.registerBuyerRoutes(protected)
	s.registerInvoiceRoutes(protected)
	s.registerTaxInvoiceRoutes(protected)
}
