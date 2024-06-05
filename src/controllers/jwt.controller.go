package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"os"
	"time"
)

var secretKey = []byte("secret-key")

func CreateToken(userUid string) (string, error) {
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
			"sub": userUid,
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

	claims, err := verifyToken(refreshToken)
	if err != nil {
		c.Writer.WriteHeader(http.StatusUnauthorized)
		_, err := fmt.Fprint(c.Writer, "Invalid token")
		if err != nil {
			return
		}
		return
	}

	userUid := claims["sub"].(string)
	newToken, err := CreateToken(userUid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"token":   newToken,
		"userUid": userUid,
	})
}
