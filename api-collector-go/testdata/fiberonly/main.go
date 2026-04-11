package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	app.Get("/status", statusHandler)
	app.Listen(":3000")
}

// statusHandler returns service status.
func statusHandler(c *fiber.Ctx) error {
	return c.JSON(map[string]string{"status": "ok"})
}
