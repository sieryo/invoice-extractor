package http

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func (s *Server) registerMiddleware() {
	s.app.Use(recover.New())
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:  "*",
		AllowMethods:  "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:  "*",
		ExposeHeaders: "Content-Length, Content-Disposition, X-File-Id, X-Exported-Filename",
	}))
}

func (s *Server) AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		sessionID := parts[1]

		sess, err := s.appCtx.AuthService.GetSession(sessionID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired session",
			})
		}

		if sess.ExpiresAt.Before(time.Now()) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "session expired",
			})
		}

		c.Locals("profileId", sess.ProfileID)
		c.Locals("userId", sess.ProfileID)
		c.Locals("sessionID", sessionID)

		return c.Next()
	}
}
