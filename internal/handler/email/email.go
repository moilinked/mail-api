// Package email 提供按 HTML 模板发信的接口。
package email

import (
	"context"
	"log/slog"
	"net/http"

	"mail-api/internal/mailer"

	"github.com/gin-gonic/gin"
)

// Queuer 接收一封待发邮件。发送由队列消费协程完成。
type Queuer interface {
	Enqueue(ctx context.Context, to string) error
}

// Handler 处理 POST /email。
type Handler struct {
	queue Queuer
}

// NewHandler 创建发信处理器。
func NewHandler(queue Queuer) *Handler {
	return &Handler{queue: queue}
}

type sendRequest struct {
	To string `json:"to"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type sendResponse struct {
	Status string `json:"status"`
}

// Post 校验收件人后只入队，不等待 SMTP 发送完成。
func (h *Handler) Post(c *gin.Context) {
	var req sendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid json"})
		return
	}

	to, err := mailer.ParseRecipient(req.To)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	if err := h.queue.Enqueue(c.Request.Context(), to); err != nil {
		slog.Error("enqueue email", "err", err, "to", to)
		c.JSON(http.StatusServiceUnavailable, errorResponse{Error: "failed to queue email"})
		return
	}

	c.JSON(http.StatusAccepted, sendResponse{Status: "queued"})
}
