package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	v1 := app.Group("/v1")
	{
		v1.Get("/users", listUsers)
		v1.Post("/users", createUser)
	}

	api := app.Group("/api")
	{
		api.Get("/items", listItems)
		api.Delete("/items/:id", deleteItem)
	}

	app.Get("/health", healthCheck)

	app.Listen(":3000")
}

// listUsers returns all users.
func listUsers(c *fiber.Ctx) error {
	c.Query("name")
	return c.JSON(map[string]interface{}{})
}

// createUser creates a new user.
func createUser(c *fiber.Ctx) error {
	var req struct{}
	_ = c.BodyParser(&req)
	return c.JSON(map[string]interface{}{})
}

// listItems returns all items.
func listItems(c *fiber.Ctx) error {
	return c.JSON(map[string]interface{}{})
}

// deleteItem removes an item by ID.
func deleteItem(c *fiber.Ctx) error {
	return c.SendStatus(204)
}

// healthCheck returns service health status.
func healthCheck(c *fiber.Ctx) error {
	return c.SendString("ok")
}
