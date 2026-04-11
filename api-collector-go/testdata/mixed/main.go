package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
)

func main() {
	// Gin routes
	r := gin.Default()
	r.GET("/gin/hello", ginHello)
	r.POST("/gin/users", ginCreateUser)

	// Echo routes
	e := echo.New()
	e.GET("/echo/hello", echoHello)
	e.DELETE("/echo/users/:id", echoDeleteUser)

	// Fiber routes
	app := fiber.New()
	app.Get("/fiber/hello", fiberHello)
	app.Put("/fiber/users/:id", fiberUpdateUser)
}

// ginHello returns a greeting from Gin.
func ginHello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "hello from gin"})
}

// ginCreateUser creates a new user via Gin.
func ginCreateUser(c *gin.Context) {
	var req struct{}
	_ = c.ShouldBindJSON(&req)
	c.JSON(http.StatusCreated, gin.H{"created": true})
}

// echoHello returns a greeting from Echo.
func echoHello(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"message": "hello from echo"})
}

// echoDeleteUser deletes a user via Echo.
func echoDeleteUser(c echo.Context) error {
	id := c.Param("id")
	return c.JSON(http.StatusOK, map[string]string{"deleted": id})
}

// fiberHello returns a greeting from Fiber.
func fiberHello(c *fiber.Ctx) error {
	return c.JSON(map[string]string{"message": "hello from fiber"})
}

// fiberUpdateUser updates a user via Fiber.
func fiberUpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(map[string]string{"updated": id})
}
