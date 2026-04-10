package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	v1 := e.Group("/v1")
	{
		v1.GET("/users", listUsers)
		v1.POST("/users", createUser)
	}

	api := e.Group("/api")
	{
		api.GET("/items", listItems)
		api.DELETE("/items/:id", deleteItem)
	}

	e.GET("/health", healthCheck)

	e.Logger.Fatal(e.Start(":8080"))
}

// listUsers returns all users.
func listUsers(c echo.Context) error {
	c.QueryParam("name")
	return c.JSON(http.StatusOK, map[string]interface{}{})
}

// createUser creates a new user.
func createUser(c echo.Context) error {
	var req struct{}
	_ = c.Bind(&req)
	return c.JSON(http.StatusCreated, map[string]interface{}{})
}

// listItems returns all items.
func listItems(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{})
}

// deleteItem removes an item by ID.
func deleteItem(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// healthCheck returns service health status.
func healthCheck(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}
