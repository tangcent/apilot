package main

import (
	"github.com/gofiber/fiber/v2"
)

type CreateUserReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	app := fiber.New()

	app.Get("/users", listUsers)
	app.Post("/users", createUser)
	app.Get("/users/:id", getUser)
	app.Put("/users/:id", updateUser)
	app.Delete("/users/:id", deleteUser)
	app.Patch("/users/:id", patchUser)
	app.Post("/upload", uploadFile)

	app.Listen(":3000")
}

// listUsers returns all users.
func listUsers(c *fiber.Ctx) error {
	name := c.Query("name")
	_ = name
	return c.JSON(map[string]interface{}{"users": []string{}})
}

// createUser creates a new user.
func createUser(c *fiber.Ctx) error {
	var req CreateUserReq
	_ = c.BodyParser(&req)
	return c.JSON(req)
}

// getUser returns a single user by ID.
func getUser(c *fiber.Ctx) error {
	id := c.Params("id")
	_ = id
	return c.JSON(map[string]interface{}{"id": id})
}

// updateUser updates an existing user.
func updateUser(c *fiber.Ctx) error {
	var req CreateUserReq
	_ = c.BodyParser(&req)
	return c.JSON(req)
}

// deleteUser removes a user by ID.
func deleteUser(c *fiber.Ctx) error {
	return c.SendStatus(204)
}

// patchUser partially updates a user.
func patchUser(c *fiber.Ctx) error {
	name := c.Query("name")
	_ = name
	return c.JSON(map[string]interface{}{"id": c.Params("id")})
}

// uploadFile handles file uploads.
func uploadFile(c *fiber.Ctx) error {
	_, _ = c.FormFile("file")
	desc := c.FormValue("description")
	_ = desc
	return c.JSON(map[string]string{"status": "ok"})
}
