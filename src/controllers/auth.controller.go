package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/yopaz-huytc/go-auth/src/config"
	"github.com/yopaz-huytc/go-auth/src/models"
	"net/http"
	"strconv"
)

func Login(c *gin.Context) {
	w := c.Writer
	r := c.Request
	w.Header().Set("Content-Type", "application/json")

	var u models.User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		fmt.Println(err) // print out the error
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userID, err := strconv.Atoi(u.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	client, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error connecting to Redis"})
		return
	}
	ctx := context.Background()

	userData := client.HGetAll(ctx, fmt.Sprintf("redis-user:%d", userID)).Val()
	if len(userData) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	fmt.Printf("User data: %v\n", userData)

	tokenString, err := CreateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user":  userData,
	})
}

func GetUserByToken(c *gin.Context) {
	w := c.Writer
	r := c.Request
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(w, "Missing authorization header")
		if err != nil {
			return
		}
		return
	}
	tokenString = tokenString[len("Bearer "):]

	claims, err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(w, "Invalid token")
		if err != nil {
			return
		}
		return
	}

	// Connect to Redis
	client, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error connecting to Redis"})
		return
	}
	ctx := context.Background()

	// Fetch user data from Redis
	userID := fmt.Sprintf("%.0f", claims["sub"].(float64))
	userData := client.HGetAll(ctx, fmt.Sprintf("redis-user:%s", userID)).Val()
	if len(userData) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Return user data in response
	c.JSON(http.StatusOK, gin.H{"user": userData})
}

func TestRedis(c *gin.Context) {
	client, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error connecting to Redis"})
		return
	}
	ctx := context.Background()

	session := map[string]string{
		"id":      "1",
		"name":    "Test",
		"surname": "Witcher",
		"company": "Redis",
		"age":     "31",
	}
	for k, v := range session {
		err := client.HSet(ctx, "redis-user:1", k, v).Err()
		if err != nil {
			panic(err)
		}
	}

	userSession := client.HGetAll(ctx, "redis-user:1").Val()
	c.JSON(200, gin.H{
		"user": userSession,
	})
}
