package http

import (
	"github.com/gofiber/fiber/v2"
	embedassets "github.com/sieryo/invoice-extractor/internal/embed"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerFrontendRoute(app *fiber.App) {
	frontendHandler := handler.NewFrontendHandler(embedassets.FrontendFS)

	frontendHandler.Init(app)
}
