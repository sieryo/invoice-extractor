package logger

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func ErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {

		var e *fiber.Error
		if errors.As(err, &e) {

			// SKIP kalau asset, menuhin log
			if e.Code == 404 && strings.HasPrefix(c.Path(), "/assets/") {
				return c.SendStatus(404)
			}

			switch {
			case e.Code >= 500:
				log.Error(
					"server error",
					"status", e.Code,
					"message", e.Message,
					"path", c.Path(),
					"method", c.Method(),
				)
			case e.Code >= 400:
				log.Warn(
					"client error",
					"status", e.Code,
					"message", e.Message,
					"path", c.Path(),
					"method", c.Method(),
				)
			default:
				log.Info(
					"http error",
					"status", e.Code,
					"message", e.Message,
				)
			}

			return c.Status(e.Code).JSON(fiber.Map{
				"error": e.Message,
			})
		}

		// fallback
		log.Error(
			"unhandled error",
			"error", err.Error(),
			"path", c.Path(),
			"method", c.Method(),
		)

		return c.Status(500).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
}
