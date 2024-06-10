package controllers

import (
	"github.com/gin-gonic/gin"
	"log"
)

// ErrorHandler is a function to handle errors
func ErrorHandler(c *gin.Context, status int, err error) {
	// Log the error
	log.Printf("Error: %v", err)

	// Send a JSON response with the error message and status code
	c.JSON(status, gin.H{
		"status": status,
		"error":  err.Error(),
	})
}
