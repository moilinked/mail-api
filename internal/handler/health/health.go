// Package health 提供服务健康检查接口。
package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 是健康检查响应。
type Response struct {
	Status string `json:"status"`
}

// Get 处理 GET /health，返回服务存活状态。
func Get(c *gin.Context) {
	c.JSON(http.StatusOK, Response{Status: "ok"})
}
