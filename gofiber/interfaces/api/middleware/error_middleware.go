package middleware

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/pkg/logger"
	"gofiber-template/pkg/utils"
)

func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		logger.Error(logger.CategoryAPI, "error_handler", "Request error occurred", err, map[string]interface{}{"status_code": code, "path": c.Path(), "method": c.Method()})

		return utils.ErrorResponse(c, code, "An error occurred", err)
	}
}