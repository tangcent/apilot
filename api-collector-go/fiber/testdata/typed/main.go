package main

import "github.com/gofiber/fiber/v2"

type BaseModel struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
}

type CreateUserReq struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required"`
	Age   *int   `json:"age"`
}

type UserVO struct {
	BaseModel
	Name   string   `json:"name"`
	Email  string   `json:"email"`
	Active bool     `json:"active"`
	Tags   []string `json:"tags"`
}

type UpdateUserReq struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type SearchQuery struct {
	Keyword string `json:"keyword" validate:"required"`
	Page    int    `json:"page"`
	Size    int    `json:"size"`
}

func main() {
	app := fiber.New()

	app.Post("/users", createUser)
	app.Get("/users/:id", getUser)
	app.Put("/users/:id", updateUser)
	app.Get("/users", listUsers)
	app.Get("/search", searchUsers)

	app.Listen(":3000")
}

// createUser creates a new user.
func createUser(c *fiber.Ctx) error {
	var req CreateUserReq
	_ = c.BodyParser(&req)
	return c.JSON(UserVO{})
}

// getUser returns a user by ID.
func getUser(c *fiber.Ctx) error {
	return c.JSON(UserVO{})
}

// updateUser updates a user.
func updateUser(c *fiber.Ctx) error {
	req := UpdateUserReq{}
	_ = c.BodyParser(&req)
	return c.JSON(UserVO{})
}

// listUsers returns all users.
func listUsers(c *fiber.Ctx) error {
	name := c.Query("name")
	_ = name
	return c.JSON(fiber.Map{"users": []UserVO{}})
}

// searchUsers searches users by query.
func searchUsers(c *fiber.Ctx) error {
	var query SearchQuery
	_ = c.QueryParser(&query)
	return c.JSON(fiber.Map{"results": []UserVO{}})
}
