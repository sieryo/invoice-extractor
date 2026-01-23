package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app"
)

type Server struct {
	app    *fiber.App
	appCtx *app.App
}

func NewServer(appCtx *app.App) *Server {
	f := fiber.New()

	s := &Server{
		app:    f,
		appCtx: appCtx,
	}

	s.registerMiddleware()
	s.registerRoutes()

	return s
}

func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}
