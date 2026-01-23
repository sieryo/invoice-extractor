package http

import "github.com/gofiber/fiber/v2/middleware/recover"

func (s *Server) registerMiddleware() {
	s.app.Use(recover.New())
}
