package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()

	r.GET("/users", listUsers)
	r.POST("/users", createUser)
	r.GET("/users/:id", getUser)
	r.PUT("/users/:id", updateUser)
	r.DELETE("/users/:id", deleteUser)
	r.PATCH("/users/:id", patchUser)
	r.HEAD("/health", healthCheck)
	r.OPTIONS("/users", userOptions)

	r.Run(":8080")
}

// listUsers returns all users.
func listUsers(c *gin.Context) {
	c.JSON(200, gin.H{"users": []string{}})
}

// createUser creates a new user.
func createUser(c *gin.Context) {
	c.JSON(201, gin.H{"id": 1})
}

// getUser returns a single user by ID.
func getUser(c *gin.Context) {
	c.JSON(200, gin.H{"id": c.Param("id")})
}

// updateUser updates an existing user.
func updateUser(c *gin.Context) {
	c.JSON(200, gin.H{"id": c.Param("id")})
}

// deleteUser removes a user by ID.
func deleteUser(c *gin.Context) {
	c.Status(204)
}

// patchUser partially updates a user.
func patchUser(c *gin.Context) {
	c.JSON(200, gin.H{"id": c.Param("id")})
}

// healthCheck returns service health status.
func healthCheck(c *gin.Context) {
	c.Status(200)
}

// userOptions returns allowed methods for /users.
func userOptions(c *gin.Context) {
	c.Status(204)
}
