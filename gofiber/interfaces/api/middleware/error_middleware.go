package middleware

import (
	"log"
	"github.com/gofiber/fiber/v2"
	"gofiber-template/pkg/utils"
)

func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		log.Printf("Error: %v", err)

		return utils.ErrorResponse(c, code, "An error occurred", err)
	}
}