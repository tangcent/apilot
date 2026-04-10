package main

import "github.com/gin-gonic/gin"

type CreateUserReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateUserReq struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

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
	r.POST("/upload", uploadFile)

	r.Run(":8080")
}

// listUsers returns all users.
func listUsers(c *gin.Context) {
	name := c.Query("name")
	role := c.DefaultQuery("role", "user")
	_ = name
	_ = role
	c.JSON(200, gin.H{"users": []string{}})
}

// createUser creates a new user.
func createUser(c *gin.Context) {
	var req CreateUserReq
	_ = c.ShouldBindJSON(&req)
	c.JSON(201, req)
}

// getUser returns a single user by ID.
func getUser(c *gin.Context) {
	c.JSON(200, gin.H{"id": c.Param("id")})
}

// updateUser updates an existing user.
func updateUser(c *gin.Context) {
	var req UpdateUserReq
	_ = c.BindJSON(&req)
	c.JSON(200, req)
}

// deleteUser removes a user by ID.
func deleteUser(c *gin.Context) {
	c.Status(204)
}

// patchUser partially updates a user.
func patchUser(c *gin.Context) {
	name := c.DefaultQuery("name", "unknown")
	_ = name
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

// uploadFile handles file uploads.
func uploadFile(c *gin.Context) {
	_, _ = c.FormFile("file")
	desc := c.PostForm("description")
	_ = desc
	c.JSON(200, gin.H{"status": "ok"})
}
