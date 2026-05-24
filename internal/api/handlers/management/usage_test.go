package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/redisqueue"
)

func TestGetUsageQueuePopsRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisqueue.SetEnabled(false)
	t.Cleanup(func() {
		redisqueue.SetEnabled(false)
	})
	redisqueue.SetEnabled(true)
	redisqueue.Enqueue([]byte(`{"model":"gpt-5.5"}`))
	redisqueue.Enqueue([]byte(`plain-record`))

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/usage-queue?count=2", nil)

	h.GetUsageQueue(ginCtx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body []any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("records = %d, want 2: %s", len(body), rec.Body.String())
	}
	if got := body[0].(map[string]any)["model"]; got != "gpt-5.5" {
		t.Fatalf("first record model = %v, want gpt-5.5", got)
	}
	if got := body[1]; got != "plain-record" {
		t.Fatalf("second record = %v, want plain-record", got)
	}
}

func TestGetUsageQueueRejectsInvalidCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, nil)
	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/usage-queue?count=0", nil)

	h.GetUsageQueue(ginCtx)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
