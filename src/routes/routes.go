package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/yopaz-huytc/go-auth/src/controllers"
)

func Routes() {
	router := gin.Default()

	router.POST("/login", controllers.Login)
	router.GET("/user-info", controllers.GetUserByToken)
	router.GET("/redis", controllers.TestRedis)
	router.POST("/refresh", controllers.RefreshToken)

	err := router.Run()
	if err != nil {
		return
	}
}
