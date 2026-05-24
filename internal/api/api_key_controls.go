package api

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	geminihandler "github.com/router-for-me/CLIProxyAPI/v6/sdk/api/handlers/gemini"
	"github.com/tidwall/gjson"
)

func (s *Server) authAndAPIKeyControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authenticateRequest(c, s.accessManager) {
			return
		}
		if !s.enforceAPIKeyControls(c) {
			return
		}
		c.Next()
	}
}

func (s *Server) enforceAPIKeyControls(c *gin.Context) bool {
	if s == nil || c == nil {
		return true
	}
	cfg := s.cfg
	if cfg == nil || len(cfg.APIKeyControls) == 0 {
		return true
	}
	apiKey := strings.TrimSpace(c.GetString("apiKey"))
	if apiKey == "" {
		return true
	}
	control := findAPIKeyControl(cfg, apiKey)
	if control == nil {
		return true
	}
	if control.Enabled != nil && !*control.Enabled {
		abortAPIKeyControl(c, http.StatusForbidden, "api_key_disabled", "API key is disabled")
		return false
	}
	if !withinAPIKeyBudget(control, usage.GetRequestStatistics()) {
		abortAPIKeyControl(c, http.StatusTooManyRequests, "api_key_budget_exceeded", "API key usage budget exceeded")
		return false
	}
	modelName, ok := extractRequestModel(c)
	if ok && modelName != "" && !apiKeyModelAllowed(control, modelName) {
		abortAPIKeyControl(c, http.StatusForbidden, "model_not_allowed", "Model is not allowed for this API key")
		return false
	}
	return true
}

func findAPIKeyControl(cfg *config.Config, apiKey string) *config.APIKeyControl {
	if cfg == nil || apiKey == "" {
		return nil
	}
	for i := range cfg.APIKeyControls {
		key := strings.TrimSpace(cfg.APIKeyControls[i].APIKey)
		if key == "" {
			key = strings.TrimSpace(cfg.APIKeyControls[i].Key)
		}
		if key == apiKey {
			return &cfg.APIKeyControls[i]
		}
	}
	return nil
}

func withinAPIKeyBudget(control *config.APIKeyControl, stats *usage.RequestStatistics) bool {
	if control == nil || control.Unlimited {
		return true
	}
	if control.MaxRequests <= 0 && control.MaxInputTokens <= 0 && control.MaxTotalTokens <= 0 && control.MaxCostUSD <= 0 {
		return true
	}
	if stats == nil {
		return true
	}
	key := strings.TrimSpace(control.APIKey)
	if key == "" {
		key = strings.TrimSpace(control.Key)
	}
	if key == "" {
		return true
	}
	snapshot := stats.Snapshot()
	apiStats, ok := snapshot.APIs[key]
	if !ok {
		return true
	}
	if control.MaxRequests > 0 && apiStats.TotalRequests >= control.MaxRequests {
		return false
	}
	if control.MaxInputTokens > 0 && apiStats.TotalInputTokens >= control.MaxInputTokens {
		return false
	}
	if control.MaxTotalTokens > 0 && apiStats.TotalTokens >= control.MaxTotalTokens {
		return false
	}
	if control.MaxCostUSD > 0 && estimateAPIKeyCostUSD(apiStats) >= control.MaxCostUSD {
		return false
	}
	return true
}

type apiKeyModelPrice struct {
	input       float64
	cachedInput float64
	output      float64
}

var apiKeyGPTModelPrices = map[string]apiKeyModelPrice{
	"gpt-5.5":                    {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.5-low-fast":           {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.5-medium-fast":        {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.5-high-fast":          {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.5-xhigh-fast":         {input: 5, cachedInput: 0.5, output: 30},
	"gpt-5.4":                    {input: 2.5, cachedInput: 0.25, output: 15},
	"gpt-5.4-mini":               {input: 0.75, cachedInput: 0.075, output: 4.5},
	"gpt-5.4-nano":               {input: 0.2, cachedInput: 0.02, output: 1.25},
	"gpt-5.3-codex":              {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark":        {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark-low":    {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark-medium": {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark-high":   {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5.3-codex-spark-xhigh":  {input: 1.75, cachedInput: 0.175, output: 14},
	"gpt-5":                      {input: 1.25, cachedInput: 0.125, output: 10},
	"gpt-5-mini":                 {input: 0.25, cachedInput: 0.025, output: 2},
	"gpt-5-nano":                 {input: 0.05, cachedInput: 0.005, output: 0.4},
	"gpt-5-pro":                  {input: 15, cachedInput: 0, output: 120},
}

var apiKeyUnknownGPTPrice = apiKeyModelPrice{input: 15, cachedInput: 0, output: 120}

func estimateAPIKeyCostUSD(apiStats usage.APISnapshot) float64 {
	var total float64
	for model, modelStats := range apiStats.Models {
		price, ok := priceForAPIKeyModel(model)
		if !ok {
			continue
		}
		for _, detail := range modelStats.Details {
			total += estimateTokenStatsCostUSD(detail.Tokens, price)
		}
	}
	return total
}

func priceForAPIKeyModel(model string) (apiKeyModelPrice, bool) {
	model = normalizeAPIKeyCostModel(model)
	if model == "" {
		return apiKeyModelPrice{}, false
	}
	if price, ok := apiKeyGPTModelPrices[model]; ok {
		return price, true
	}
	if strings.HasPrefix(model, "gpt-") {
		return apiKeyUnknownGPTPrice, true
	}
	return apiKeyModelPrice{}, false
}

func normalizeAPIKeyCostModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(model, "models/")))
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		model = strings.TrimSpace(model[idx+1:])
	}
	return strings.TrimPrefix(model, "models/")
}

func estimateTokenStatsCostUSD(tokens usage.TokenStats, price apiKeyModelPrice) float64 {
	inputTokens := clampNonNegative(tokens.InputTokens)
	cachedTokens := clampNonNegative(tokens.CachedTokens)
	if cachedTokens > inputTokens {
		cachedTokens = inputTokens
	}
	uncachedInputTokens := inputTokens - cachedTokens
	outputTokens := clampNonNegative(tokens.OutputTokens)
	if outputTokens == 0 && tokens.TotalTokens > inputTokens {
		outputTokens = tokens.TotalTokens - inputTokens
	}
	return (float64(uncachedInputTokens)*price.input +
		float64(cachedTokens)*price.cachedInput +
		float64(outputTokens)*price.output) / 1_000_000
}

func clampNonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func extractRequestModel(c *gin.Context) (string, bool) {
	if c == nil || c.Request == nil {
		return "", false
	}
	if model := extractGeminiModelFromPath(c.Request.URL.Path); model != "" {
		return model, true
	}
	if queryModel := strings.TrimSpace(c.Query("model")); queryModel != "" {
		return queryModel, true
	}
	if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch {
		return "", false
	}
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if contentType != "" && !strings.Contains(contentType, "json") {
		return "", false
	}
	if c.Request.Body == nil {
		return "", false
	}
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Request.Body = io.NopCloser(bytes.NewReader(nil))
		return "", false
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))
	if len(bytes.TrimSpace(rawBody)) == 0 {
		return "", false
	}
	modelResult := gjson.GetBytes(rawBody, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String {
		return "", false
	}
	return strings.TrimSpace(modelResult.String()), true
}

func extractGeminiModelFromPath(path string) string {
	const marker = "/models/"
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	model := path[idx+len(marker):]
	if model == "" {
		return ""
	}
	if cut := strings.IndexAny(model, ":/?#"); cut >= 0 {
		model = model[:cut]
	}
	return strings.TrimPrefix(strings.TrimSpace(model), "models/")
}

func (s *Server) filterModelsForAPIKey(c *gin.Context, models []map[string]any) []map[string]any {
	if len(models) == 0 || s == nil || s.cfg == nil || c == nil {
		return models
	}
	control := findAPIKeyControl(s.cfg, strings.TrimSpace(c.GetString("apiKey")))
	if control == nil {
		return models
	}
	filtered := make([]map[string]any, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		if apiKeyModelAllowed(control, modelNameFromMap(model)) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func (s *Server) geminiModelsHandler(handler *geminihandler.GeminiAPIHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if handler == nil {
			c.JSON(http.StatusOK, gin.H{"models": []map[string]any{}})
			return
		}
		rawModels := s.filterModelsForAPIKey(c, handler.Models())
		normalizedModels := make([]map[string]any, 0, len(rawModels))
		defaultMethods := []string{"generateContent"}
		for _, model := range rawModels {
			normalizedModel := make(map[string]any, len(model))
			for k, v := range model {
				normalizedModel[k] = v
			}
			if name, ok := normalizedModel["name"].(string); ok && name != "" {
				if !strings.HasPrefix(name, "models/") {
					normalizedModel["name"] = "models/" + name
				}
				if displayName, _ := normalizedModel["displayName"].(string); displayName == "" {
					normalizedModel["displayName"] = name
				}
				if description, _ := normalizedModel["description"].(string); description == "" {
					normalizedModel["description"] = name
				}
			}
			if _, ok := normalizedModel["supportedGenerationMethods"]; !ok {
				normalizedModel["supportedGenerationMethods"] = defaultMethods
			}
			normalizedModels = append(normalizedModels, normalizedModel)
		}
		c.JSON(http.StatusOK, gin.H{"models": normalizedModels})
	}
}

func modelNameFromMap(model map[string]any) string {
	for _, key := range []string{"id", "name"} {
		value, ok := model[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimPrefix(strings.TrimSpace(value), "models/")
		}
	}
	return ""
}

func apiKeyModelAllowed(control *config.APIKeyControl, model string) bool {
	if control == nil {
		return true
	}
	model = strings.TrimPrefix(strings.TrimSpace(model), "models/")
	if model == "" {
		return true
	}
	for _, pattern := range control.ExcludedModels {
		if modelPatternMatches(model, pattern) {
			return false
		}
	}
	if len(control.Models) == 0 {
		return true
	}
	for _, pattern := range control.Models {
		if modelPatternMatches(model, pattern) {
			return true
		}
	}
	return false
}

func modelPatternMatches(model, pattern string) bool {
	model = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(model, "models/")))
	pattern = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(pattern, "models/")))
	if model == "" || pattern == "" {
		return false
	}
	if pattern == "*" || pattern == model {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return false
	}
	parts := strings.Split(pattern, "*")
	position := 0
	if parts[0] != "" {
		if !strings.HasPrefix(model, parts[0]) {
			return false
		}
		position = len(parts[0])
	}
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			continue
		}
		next := strings.Index(model[position:], part)
		if next < 0 {
			return false
		}
		position += next + len(part)
	}
	last := parts[len(parts)-1]
	return last == "" || strings.HasSuffix(model, last)
}

func abortAPIKeyControl(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "api_key_access_error",
			"code":    code,
		},
	})
}
