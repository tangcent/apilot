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
}

type UserVO struct {
	BaseModel
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUserReq struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

func main() {
	e := echo.New()

	e.GET("/users", listUsers)
	e.POST("/users", createUser)
	e.GET("/users/:id", getUser)
	e.PUT("/users/:id", updateUser)
	e.DELETE("/users/:id", deleteUser)
	e.PATCH("/users/:id", patchUser)
	e.POST("/upload", uploadFile)

	e.Logger.Fatal(e.Start(":8080"))
}

// listUsers returns all users.
func listUsers(c echo.Context) error {
	name := c.QueryParam("name")
	_ = name
	return c.JSON(http.StatusOK, map[string]interface{}{"users": []string{}})
}

// createUser creates a new user.
func createUser(c echo.Context) error {
	var req CreateUserReq
	_ = c.Bind(&req)
	return c.JSON(http.StatusCreated, UserVO{})
}

// getUser returns a single user by ID.
func getUser(c echo.Context) error {
	id := c.Param("id")
	_ = id
	return c.JSON(http.StatusOK, UserVO{})
}

// updateUser updates an existing user.
func updateUser(c echo.Context) error {
	var req UpdateUserReq
	_ = c.Bind(&req)
	return c.JSON(http.StatusOK, UserVO{})
}

// deleteUser removes a user by ID.
func deleteUser(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// patchUser partially updates a user.
func patchUser(c echo.Context) error {
	name := c.QueryParam("name")
	_ = name
	return c.JSON(http.StatusOK, map[string]interface{}{"id": c.Param("id")})
}

// uploadFile handles file uploads.
func uploadFile(c echo.Context) error {
	_, _ = c.FormFile("file")
	desc := c.FormValue("description")
	_ = desc
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
