package handler

import (
	"github.com/gofiber/fiber/v2"
)

type SuccessResponse struct {
	Code    int    `json:"code"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

func SendSuccess(c *fiber.Ctx, statusCode int, data interface{}, message string) error {
	return c.Status(statusCode).JSON(SuccessResponse{
		Code:    statusCode,
		Data:    data,
		Message: message,
	})
}

func SendError(c *fiber.Ctx, statusCode int, errorMessage string) error {
	return c.Status(statusCode).JSON(ErrorResponse{
		Code:  statusCode,
		Error: errorMessage,
	})
}
