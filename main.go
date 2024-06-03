package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	redisHost string
	redisPort string
	redisPass string
	port      string
	appUrl    string

	secretKey = []byte("secret-key")
)

type User struct {
	ID      string `json:"id" validate:"required"`
	Name    string `json:"name" validate:"required"`
	Surname string `json:"surname" validate:"required"`
	Company string `json:"company" validate:"required"`
	Age     string `json:"age" validate:"required"`
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	port = os.Getenv("PORT")

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(200, "Hello, you've requested: %s\n", c.Request.URL.Path)
	})
	router.POST("/login", Login)
	router.GET("/test", TestProtected)
	router.GET("/redis", TestRedis)

	err = router.Run(":" + port)
	if err != nil {
		return
	}
}

func Login(c *gin.Context) {
	w := c.Writer
	r := c.Request
	w.Header().Set("Content-Type", "application/json")

	var u User
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

	redisHost = os.Getenv("REDIS_HOST")
	redisPort = os.Getenv("REDIS_PORT")
	redisPass = os.Getenv("REDIS_PASS")

	client := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPass,
		DB:       0,
	})

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

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func TestProtected(c *gin.Context) {
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

	err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(w, "Invalid token")
		if err != nil {
			return
		}
		return
	}
	_, err = fmt.Fprint(w, "Welcome to the the protected area")
	if err != nil {
		return
	}
}

func TestRedis(c *gin.Context) {
	redisHost = os.Getenv("REDIS_HOST")
	redisPort = os.Getenv("REDIS_PORT")
	redisPass = os.Getenv("REDIS_PASS")

	client := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort,
		Password: redisPass,
		DB:       0,
	})

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

func CreateToken(userID int) (string, error) {
	jti := make([]byte, 16)
	_, err := rand.Read(jti)
	if err != nil {
		return "", err
	}
	appUrl = os.Getenv("APP_URL")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"iss": appUrl,
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(time.Hour * 24).Unix(),
			"nbf": time.Now().Unix(),
			"jti": hex.EncodeToString(jti),
			"sub": userID,
			"prv": "23bd5c8949f600adb39e701c400872db7a5976f7",
		})
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", nil
	}
	return tokenString, nil
}

func verifyToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return fmt.Errorf("invalid token")
	}
	return nil
}
