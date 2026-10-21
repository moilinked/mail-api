package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		wantStatus int
		wantBody   string
	}{
		{name: "alive", wantStatus: http.StatusOK, wantBody: "ok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.GET("/health", Get)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			var got Response
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got.Status != tt.wantBody {
				t.Fatalf("status field = %q, want %q", got.Status, tt.wantBody)
			}
		})
	}
}
