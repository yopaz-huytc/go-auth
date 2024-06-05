package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/yopaz-huytc/go-auth/src/config"
	"github.com/yopaz-huytc/go-auth/src/models"
	"net/http"
)

func Login(c *gin.Context) {
	var u models.User
	// Parse the request body
	err := json.NewDecoder(c.Request.Body).Decode(&u)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userUid := u.UID
	if userUid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UID is required"})
		return
	}

	client, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error connecting to Redis"})
		return
	}
	ctx := context.Background()

	userData, err := fetchUserData(ctx, client, userUid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tokenString, err := CreateToken(userUid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User logged in successfully",
		"token":   tokenString,
		"user":    userData,
	})
}

func GetUserByToken(c *gin.Context) {
	tokenString := c.Request.Header.Get("Authorization")
	if tokenString == "" {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(c.Writer, "Missing authorization header")
		if err != nil {
			return
		}
		return
	}
	tokenString = tokenString[len("Bearer "):]

	claims, err := verifyToken(tokenString)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(c.Writer, "Invalid token")
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
	userUid := claims["sub"].(string)
	fmt.Printf("User UID: %s\n", userUid)
	userData := client.HGetAll(ctx, fmt.Sprintf("geomark-user:%s", userUid)).Val()
	if len(userData) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Return user data in response
	c.JSON(http.StatusOK, gin.H{
		"message": "User data fetched successfully",
		"user":    userData,
	})
}

func fetchUserData(ctx context.Context, client *redis.Client, userUid string) (map[string]string, error) {
	userData := client.HGetAll(ctx, fmt.Sprintf("geomark-user:%s", userUid)).Val()
	if len(userData) == 0 {
		// Fetch user data from the database
		var db = config.ConnectDB()
		// close the database connection after the function returns
		defer config.DisconnectDB(db)
		user, err := models.GetUserByUID(db, userUid)
		if err != nil {
			return nil, fmt.Errorf("error fetching user data from the database: %w", err)
		}
		userData = map[string]string{
			"id":          fmt.Sprint(user.ID),
			"name":        user.Name,
			"email":       user.Email,
			"description": user.Description,
			"uid":         user.UID,
			"parent_id":   fmt.Sprint(user.ParentId),
		}
		for k, v := range userData {
			err := client.HSet(ctx, fmt.Sprintf("geomark-user:%s", userUid), k, v).Err()
			if err != nil {
				return nil, fmt.Errorf("error setting user data in Redis: %w", err)
			}
		}
	}
	return userData, nil
}
