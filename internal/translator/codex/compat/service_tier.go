package compat

import (
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// NormalizeServiceTier rewrites client-facing Codex service tiers to the wire value
// currently accepted by the Codex upstream. Unknown values are removed.
func NormalizeServiceTier(rawJSON []byte) []byte {
	tierResult := gjson.GetBytes(rawJSON, "service_tier")
	if !tierResult.Exists() {
		return rawJSON
	}
	tier, ok := NormalizeServiceTierValue(tierResult.String())
	if !ok {
		rawJSON, _ = sjson.DeleteBytes(rawJSON, "service_tier")
		return rawJSON
	}
	rawJSON, _ = sjson.SetBytes(rawJSON, "service_tier", tier)
	return rawJSON
}

// NormalizeServiceTierValue returns the Codex wire value for a downstream service tier.
func NormalizeServiceTierValue(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "priority", "fast":
		return "priority", true
	default:
		return "", false
	}
}
