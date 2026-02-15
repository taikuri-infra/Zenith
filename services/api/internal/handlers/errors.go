package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(APIError{
		Code:    code,
		Message: message,
	})
}

func NewBadRequest(detail string) *fiber.Error {
	return fiber.NewError(fiber.StatusBadRequest, detail)
}

func NewNotFound(resource string) *fiber.Error {
	return fiber.NewError(fiber.StatusNotFound, resource+" not found")
}

func NewUnauthorized(detail string) *fiber.Error {
	return fiber.NewError(fiber.StatusUnauthorized, detail)
}

func NewForbidden(detail string) *fiber.Error {
	return fiber.NewError(fiber.StatusForbidden, detail)
}

func NewConflict(detail string) *fiber.Error {
	return fiber.NewError(fiber.StatusConflict, detail)
}
