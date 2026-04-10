package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	
	r.GET("/users", listUsers)
	r.POST("/users", createUser)
	r.GET("/users/:id", getUser)
	r.PUT("/users/:id", updateUser)
	r.DELETE("/users/:id", deleteUser)
	
	r.Run(":8080")
}

func listUsers(c *gin.Context) {
	c.JSON(200, gin.H{"users": []string{}})
}

func createUser(c *gin.Context) {
	c.JSON(201, gin.H{"id": 1})
}

func getUser(c *gin.Context) {
	c.JSON(200, gin.H{"id": c.Param("id")})
}

func updateUser(c *gin.Context) {
	c.JSON(200, gin.H{"id": c.Param("id")})
}

func deleteUser(c *gin.Context) {
	c.Status(204)
}
