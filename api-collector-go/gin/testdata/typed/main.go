package main

import "github.com/gin-gonic/gin"

type BaseModel struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
}

type CreateUserReq struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
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
	r := gin.Default()

	r.POST("/users", createUser)
	r.GET("/users/:id", getUser)
	r.PUT("/users/:id", updateUser)
	r.GET("/users", listUsers)

	r.Run(":8080")
}

// createUser creates a new user.
func createUser(c *gin.Context) {
	var req CreateUserReq
	_ = c.ShouldBindJSON(&req)
	c.JSON(201, UserVO{})
}

// getUser returns a user by ID.
func getUser(c *gin.Context) {
	c.JSON(200, UserVO{})
}

// updateUser updates a user.
func updateUser(c *gin.Context) {
	req := UpdateUserReq{}
	_ = c.ShouldBindJSON(&req)
	c.JSON(200, UserVO{})
}

// listUsers returns all users.
func listUsers(c *gin.Context) {
	name := c.Query("name")
	_ = name
	c.JSON(200, gin.H{"users": []UserVO{}})
}
