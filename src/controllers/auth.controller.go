package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/yopaz-huytc/go-auth/src/config"
	"github.com/yopaz-huytc/go-auth/src/models"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

func Login(c *gin.Context) {
	var u models.User
	// Parse the request body
	err := json.NewDecoder(c.Request.Body).Decode(&u)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userId := u.ID
	// if the user ID is empty, return an error
	if userId == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	client, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error connecting to Redis"})
		return
	}
	ctx := context.Background()

	userData, err := fetchUserData(ctx, client, int(userId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tokenString, err := CreateToken(int(userId), 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating token"})
		return
	}

	err = client.SAdd(ctx, "whiteListToken", tokenString).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding token to whitelist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User logged in successfully",
		"token":   tokenString,
		"user":    userData,
	})
}

func GetUserByToken(c *gin.Context) {
	// Connect to Redis
	client, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error connecting to Redis"})
		return
	}
	tokenString := c.Request.Header.Get("Authorization")
	ctx := context.Background()

	if tokenString == "" {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(c.Writer, "Missing authorization header")
		if err != nil {
			return
		}
		return
	}
	tokenString = tokenString[len("Bearer "):]

	oldToken := tokenString

	// Check if the old token is in the whitelist
	isMember := client.SIsMember(ctx, "whiteListToken", oldToken).Val()
	if !isMember {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is not in the whitelist"})
		return
	}

	claims, err := verifyToken(tokenString)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(c.Writer, "Invalid token")
		if err != nil {
			return
		}
		return
	}

	// Fetch user data from Redis
	userIdFloat := claims["sub"].(float64)
	userId := int(userIdFloat)
	userData := client.HGetAll(ctx, fmt.Sprintf("geomark-user:%d", userId)).Val()
	if len(userData) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	ownerProjectUids, err := getUserManagerProject(client, userId, "owner:project_uids")
	sharedProjectUids, err := getUserManagerProject(client, userId, "shared:project_uids")
	managedProjectUids, err := getUserManagerProject(client, userId, "managed:project_uids")

	userData["owner_project_uids"] = fmt.Sprintf("%v", ownerProjectUids)
	userData["shared_project_uids"] = fmt.Sprintf("%v", sharedProjectUids)
	userData["managed_project_uids"] = fmt.Sprintf("%v", managedProjectUids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching cache value"})
		return
	}
	// Return user data in response
	c.JSON(http.StatusOK, gin.H{
		"status":  http.StatusOK,
		"message": "User data fetched successfully",
		"data":    userData,
	})
}

func fetchUserData(ctx context.Context, client *redis.Client, userId int) (map[string]string, error) {
	userData := client.HGetAll(ctx, fmt.Sprintf("geomark-user:%d", userId)).Val()
	if len(userData) == 0 {
		// Fetch user data from the database
		var db = config.ConnectDB()
		// close the database connection after the function returns
		defer config.DisconnectDB(db)
		user, err := models.GetUserById(db, userId)
		if err != nil {
			return nil, fmt.Errorf("error fetching user data from the database: %w", err)
		}
		userData = map[string]string{
			"id":                fmt.Sprint(user.ID),
			"name":              user.Name,
			"email":             user.Email,
			"description":       user.Description,
			"uid":               user.UID,
			"parent_id":         fmt.Sprint(user.ParentId),
			"email_verified_at": user.EmailVerifiedAt,
			"created_at":        user.CreatedAt,
			"updated_at":        user.UpdatedAt,
			"deleted_at":        user.DeletedAt,
		}
		for k, v := range userData {
			err := client.HSet(ctx, fmt.Sprintf("geomark-user:%d", userId), k, v).Err()
			if err != nil {
				return nil, fmt.Errorf("error setting user data in Redis: %w", err)
			}
		}

		err = client.Expire(ctx, fmt.Sprintf("geomark-user:%d", userId), 15*time.Minute).Err()
		if err != nil {
			return nil, fmt.Errorf("error setting key expiration in Redis: %w", err)
		}
	}
	return userData, nil
}

func getUserManagerProject(client *redis.Client, userId int, redisKey string) ([]string, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local"
	}
	cacheKey := fmt.Sprintf("geomark_database_geomark_cache_:denso:%s:%d:%s", appEnv, userId, redisKey)
	ctx := context.Background()
	val, err := client.Get(ctx, cacheKey).Result()
	if err != nil {
		fmt.Println(err)
	}
	cacheValue, err := DeserializedCacheValue(val)
	if err != nil {
		return nil, fmt.Errorf("error deserializing cache value: %w", err)
	}
	return cacheValue, nil
}

func DeserializedCacheValue(cacheValue string) ([]string, error) {
	regex := regexp.MustCompile(`i:(\d+);s:\d+:"([^"]+)";`)
	matches := regex.FindAllStringSubmatch(cacheValue, -1)

	if matches == nil {
		fmt.Println("Invalid serialized data")
		return nil, nil
	}

	var array []string
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		value := match[2]
		array = append(array, value)
	}

	return array, nil
}

func Logout(c *gin.Context) {
	// Get the token from the Authorization header
	tokenString := c.Request.Header.Get("Authorization")
	fmt.Println("Token: ", tokenString)
	if tokenString == "" {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(c.Writer, "Missing authorization header")
		if err != nil {
			log.Printf("Error writing to response: %v", err)
			return
		}
		return
	}
	oldToken := tokenString[len("Bearer "):]

	// Connect to Redis
	client, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error connecting to Redis"})
		return
	}
	ctx := context.Background()

	// Check if the old token is in the whitelist
	isMember := client.SIsMember(ctx, "whiteListToken", oldToken).Val()
	if !isMember {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is not in the whitelist"})
		return
	}

	// Remove the token from the whitelist
	err = client.SRem(ctx, "whiteListToken", oldToken).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error removing token from whitelist"})
		return
	}

	// Return a success message
	c.JSON(http.StatusOK, gin.H{
		"message": "User logged out successfully",
	})
}
