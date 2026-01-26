package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app"
	"github.com/sieryo/invoice-extractor/internal/pkg/logger"
)

type Server struct {
	app    *fiber.App
	appCtx *app.App
}

func NewServer(appCtx *app.App) *Server {
	f := fiber.New(fiber.Config{
		BodyLimit:    50 * 1024 * 1024,
		ErrorHandler: logger.ErrorHandler(appCtx.Logger),
	})

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
