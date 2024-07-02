package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/langgenius/dify-plugin-daemon/internal/types/app"
)

func server(config *app.Config) {
	engine := gin.Default()

	engine.GET("/health/check", HealthCheck)
	engine.POST("/plugin/invoke", CheckingKey(config.DifyCallingKey), InvokePlugin)

	engine.Run(fmt.Sprintf(":%d", config.DifyCallingPort))
}
