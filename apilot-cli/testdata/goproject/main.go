package main

import "github.com/gin-gonic/gin"

// User is a registered account.
type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   *int   `json:"age"`
}

// CreateUserReq is the payload accepted by POST /users.
type CreateUserReq struct {
	Name     string   `json:"name" binding:"required"`
	Email    string   `json:"email" binding:"required"`
	Password string   `json:"password" binding:"required"`
	Tags     []string `json:"tags"`
}

// UpdateUserReq is the payload accepted by PUT /users/:id.
type UpdateUserReq struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

// ErrorResponse is returned for every failed request.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	r := gin.Default()

	r.GET("/users", listUsers)
	r.POST("/users", createUser)
	r.GET("/users/:id", getUser)
	r.PUT("/users/:id", updateUser)
	r.DELETE("/users/:id", deleteUser)

	_ = r.Run(":8080")
}

// listUsers returns a filtered page of users.
func listUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	page := c.DefaultQuery("page", "1")
	_ = keyword
	_ = page
	c.JSON(200, gin.H{"users": []User{}})
}

// createUser registers a new account.
func createUser(c *gin.Context) {
	var req CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(201, User{})
}

// getUser returns a single user by id.
func getUser(c *gin.Context) {
	_ = c.GetHeader("X-Request-Id")
	c.JSON(200, User{})
}

// updateUser updates an existing user.
func updateUser(c *gin.Context) {
	var req UpdateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, ErrorResponse{Code: 400, Message: err.Error()})
		return
	}
	c.JSON(200, User{})
}

// deleteUser removes a user by id.
func deleteUser(c *gin.Context) {
	c.Status(204)
}
