package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"os"
	"strconv"
	"time"
)

var secretKey = []byte("secret-key")

func CreateToken(userID int) (string, error) {
	appUrl := os.Getenv("APP_URL")
	jti := make([]byte, 16)
	_, err := rand.Read(jti)
	if err != nil {
		return "", err
	}
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

func verifyToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
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
	w := c.Writer
	r := c.Request
	w.Header().Set("Content-Type", "application/json")

	refreshToken := r.Header.Get("Authorization")
	if refreshToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(w, "Missing authorization header")
		if err != nil {
			return
		}
		return
	}
	refreshToken = refreshToken[len("Bearer "):]

	claims, err := verifyToken(refreshToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(w, "Invalid token")
		if err != nil {
			return
		}
		return
	}

	userID := fmt.Sprintf("%.0f", claims["sub"].(float64))

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error converting user ID to integer"})
		return
	}

	newToken, err := CreateToken(userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newToken})
}
