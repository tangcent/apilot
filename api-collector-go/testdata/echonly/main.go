package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	e.GET("/health", healthCheck)
	e.Logger.Fatal(e.Start(":8080"))
}

// healthCheck returns service health.
func healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
