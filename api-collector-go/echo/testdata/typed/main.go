package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

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

func main() {
	e := echo.New()

	e.POST("/users", createUser)
	e.GET("/users/:id", getUser)
	e.PUT("/users/:id", updateUser)
	e.GET("/users", listUsers)

	e.Logger.Fatal(e.Start(":8080"))
}

// createUser creates a new user.
func createUser(c echo.Context) error {
	var req CreateUserReq
	_ = c.Bind(&req)
	return c.JSON(http.StatusCreated, UserVO{})
}

// getUser returns a user by ID.
func getUser(c echo.Context) error {
	return c.JSON(http.StatusOK, UserVO{})
}

// updateUser updates a user.
func updateUser(c echo.Context) error {
	req := UpdateUserReq{}
	_ = c.Bind(&req)
	return c.JSON(http.StatusOK, UserVO{})
}

// listUsers returns all users.
func listUsers(c echo.Context) error {
	name := c.QueryParam("name")
	_ = name
	return c.JSON(http.StatusOK, map[string]interface{}{"users": []UserVO{}})
}
