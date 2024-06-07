package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/yopaz-huytc/go-auth/src/config"
	"net/http"
	"os"
	"time"
)

type RequestBody struct {
	TTL int `json:"ttl"`
}

var jwtKey = []byte("secret_key")

func CreateToken(userId int, ttl int) (string, error) {
	if ttl == 0 {
		ttl = 24 * 90
	}
	fmt.Println("TTL of Token is: ", ttl, " hours")
	appUrl := os.Getenv("APP_URL")
	secretKey := os.Getenv("JWT_SECRET")
	jwtKey = []byte(secretKey)
	jti := make([]byte, 16)
	_, err := rand.Read(jti)
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"iss": appUrl,
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(time.Hour * time.Duration(ttl)).Unix(),
			"nbf": time.Now().Unix(),
			"jti": hex.EncodeToString(jti),
			"sub": userId,
		})
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", nil
	}
	return tokenString, nil
}

func verifyToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {

		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return token.Claims.(jwt.MapClaims), nil
}

func RefreshToken(c *gin.Context) {
	var reqBody RequestBody
	if err := c.BindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	ttl := reqBody.TTL
	if ttl == 0 {
		ttl = 24 * 90
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	refreshToken := c.Request.Header.Get("Authorization")
	if refreshToken == "" {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(c.Writer, "Missing authorization header")
		if err != nil {
			return
		}
		return
	}
	refreshToken = refreshToken[len("Bearer "):]

	oldToken := refreshToken

	client, err := config.ConnectRedis()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error connecting to Redis"})
		return
	}
	ctx := context.Background()
	// Check if the old token is in the whitelist
	fmt.Println("Old Token: ", oldToken)
	isMember := client.SIsMember(ctx, "whiteListToken", oldToken).Val()
	if !isMember {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is not in the whitelist"})
		return
	}

	claims, err := verifyToken(refreshToken)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(c.Writer, "Invalid token")
		if err != nil {
			return
		}
		return
	}

	userId := claims["sub"].(float64)
	newToken, err := CreateToken(int(userId), ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating token"})
		return
	}

	// Remove the old token from the whitelist and add the new one
	err = client.SRem(ctx, "whiteListToken", oldToken).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error removing old token from whitelist"})
		return
	}

	err = client.SAdd(ctx, "whiteListToken", newToken).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding new token to whitelist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"token":   newToken,
		"userUid": userId,
	})
}
