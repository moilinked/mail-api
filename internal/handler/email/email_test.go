package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeQueue struct {
	to  string
	err error
}

func (f *fakeQueue) Enqueue(_ context.Context, to string) error {
	f.to = to
	return f.err
}

func TestPost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		queueErr   error
		wantStatus int
		wantTo     string
		wantError  string
	}{
		{
			name:       "queues email",
			body:       `{"to":"User <user@example.com>"}`,
			wantStatus: http.StatusAccepted,
			wantTo:     "user@example.com",
		},
		{
			name:       "rejects invalid json",
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid json",
		},
		{
			name:       "rejects missing recipient",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "recipient is required",
		},
		{
			name:       "rejects invalid recipient",
			body:       `{"to":"not-an-email"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid recipient",
		},
		{
			name:       "reports queue failure",
			body:       `{"to":"user@example.com"}`,
			queueErr:   errors.New("queue closed"),
			wantStatus: http.StatusServiceUnavailable,
			wantTo:     "user@example.com",
			wantError:  "failed to queue email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue := &fakeQueue{err: tt.queueErr}
			engine := gin.New()
			engine.POST("/email", NewHandler(queue).Post)

			req := httptest.NewRequest(http.MethodPost, "/email", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if queue.to != tt.wantTo {
				t.Fatalf("recipient = %q, want %q", queue.to, tt.wantTo)
			}
			if tt.wantError == "" {
				var got sendResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if got.Status != "queued" {
					t.Fatalf("status field = %q", got.Status)
				}
				return
			}

			var got errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got.Error != tt.wantError {
				t.Fatalf("error = %q, want %q", got.Error, tt.wantError)
			}
		})
	}
}
