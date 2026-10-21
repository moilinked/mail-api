// Package router 注册 HTTP 路由。
package router

import (
	"mail-api/internal/handler/health"

	"github.com/gin-gonic/gin"
)

// Setup 把路由挂到 engine 上。sendEmail 处理 POST /emails。
func Setup(engine *gin.Engine, sendEmail gin.HandlerFunc) {
	engine.GET("/health", health.Get)
	engine.POST("/emails", sendEmail)
}
