package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()

	v1 := r.Group("/v1")
	{
		v1.GET("/users", listUsers)
		v1.POST("/users", createUser)
	}

	api := r.Group("/api")
	{
		api.GET("/items", listItems)
		api.DELETE("/items/:id", deleteItem)
	}

	r.GET("/health", healthCheck)

	r.Run(":8080")
}

// listUsers returns all users.
func listUsers(c *gin.Context) {
	c.JSON(200, gin.H{})
}

// createUser creates a new user.
func createUser(c *gin.Context) {
	c.JSON(201, gin.H{})
}

// listItems returns all items.
func listItems(c *gin.Context) {
	c.JSON(200, gin.H{})
}

// deleteItem removes an item by ID.
func deleteItem(c *gin.Context) {
	c.Status(204)
}

// healthCheck returns service health status.
func healthCheck(c *gin.Context) {
	c.Status(200)
}
