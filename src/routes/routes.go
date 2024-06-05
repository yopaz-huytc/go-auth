package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/yopaz-huytc/go-auth/src/controllers"
)

func Routes() {
	router := gin.Default()
	router.Use(JSONMiddleware())

	router.POST("/login", controllers.Login)
	router.GET("/user-info", controllers.GetUserByToken)
	router.POST("/refresh", controllers.RefreshToken)

	err := router.Run()
	if err != nil {

		return
	}
}

func JSONMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Next()
	}
}
