package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	proxyconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestAPIKeyControl_AllowsModelAndPreservesJSONBody(t *testing.T) {
	server := newTestServer(t)
	server.cfg.APIKeyControls = []proxyconfig.APIKeyControl{
		{APIKey: "test-key", Models: []string{"allowed-*"}},
	}
	server.engine.POST("/test/api-key-control", server.authAndAPIKeyControlMiddleware(), func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil {
			t.Fatalf("handler failed to read body: %v", err)
		}
		if !strings.Contains(string(body), `"model":"allowed-model"`) {
			t.Fatalf("request body was not preserved: %s", string(body))
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test/api-key-control", strings.NewReader(`{"model":"allowed-model","messages":[]}`))
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
}

func TestAPIKeyControl_BlocksDisallowedModel(t *testing.T) {
	server := newTestServer(t)
	server.cfg.APIKeyControls = []proxyconfig.APIKeyControl{
		{APIKey: "test-key", Models: []string{"allowed-*"}},
	}
	server.engine.POST("/test/api-key-control-denied", server.authAndAPIKeyControlMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test/api-key-control-denied", strings.NewReader(`{"model":"blocked-model","messages":[]}`))
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.engine.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "model_not_allowed") {
		t.Fatalf("response missing model_not_allowed code: %s", rr.Body.String())
	}
}

func TestFilterModelsForAPIKey(t *testing.T) {
	server := &Server{cfg: &proxyconfig.Config{
		APIKeyControls: []proxyconfig.APIKeyControl{
			{APIKey: "limited-key", Models: []string{"gpt-5.3-codex-spark*"}},
		},
	}}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("apiKey", "limited-key")

	models := []map[string]any{
		{"id": "gpt-5.3-codex-spark-high", "object": "model"},
		{"id": "gpt-5.5-codex", "object": "model"},
	}
	filtered := server.filterModelsForAPIKey(ctx, models)
	if len(filtered) != 1 {
		t.Fatalf("filtered models = %d, want 1: %#v", len(filtered), filtered)
	}
	if got := filtered[0]["id"]; got != "gpt-5.3-codex-spark-high" {
		t.Fatalf("filtered model = %v, want spark model", got)
	}
}

func TestAPIKeyControl_AllowsEstimatedCostBelowLimit(t *testing.T) {
	stats := usage.NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		APIKey: "cost-key",
		Model:  "gpt-5.5-low-fast",
		Detail: coreusage.Detail{
			InputTokens:  100_000,
			CachedTokens: 20_000,
			OutputTokens: 100_000,
			TotalTokens:  200_000,
		},
	})

	control := &proxyconfig.APIKeyControl{APIKey: "cost-key", MaxCostUSD: 30}
	if !withinAPIKeyBudget(control, stats) {
		t.Fatal("withinAPIKeyBudget() = false, want true below estimated cost limit")
	}
}

func TestAPIKeyControl_BlocksEstimatedCostAtLimit(t *testing.T) {
	stats := usage.NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		APIKey: "cost-key",
		Model:  "gpt-5.5-low-fast",
		Detail: coreusage.Detail{
			InputTokens:  1_000_000,
			CachedTokens: 200_000,
			OutputTokens: 1_000_000,
			TotalTokens:  2_000_000,
		},
	})

	control := &proxyconfig.APIKeyControl{APIKey: "cost-key", MaxCostUSD: 30}
	if withinAPIKeyBudget(control, stats) {
		t.Fatal("withinAPIKeyBudget() = true, want false at estimated cost limit")
	}
}
