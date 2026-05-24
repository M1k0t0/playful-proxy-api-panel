package chat_completions

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestConvertOpenAIRequestToGemini_StripsUnsupportedSchemaFields(t *testing.T) {
	inputJSON := []byte(`{
		"model": "gemini-3-pro-preview",
		"messages": [{"role": "user", "content": "hello"}],
		"tools": [{
			"type": "function",
			"function": {
				"name": "lookup",
				"description": "Lookup",
				"parameters": {
					"type": "object",
					"$comment": "must not be forwarded",
					"properties": {
						"mode": {
							"type": "string",
							"enum": ["a", "b"],
							"enumDescriptions": ["A", "B"],
							"$comment": "nested"
						}
					}
				}
			}
		}]
	}`)

	output := ConvertOpenAIRequestToGemini("gemini-3-pro-preview", inputJSON, false)
	outputText := string(output)
	if strings.Contains(outputText, "$comment") || strings.Contains(outputText, "enumDescriptions") {
		t.Fatalf("unsupported schema fields were forwarded: %s", outputText)
	}
	if got := gjson.GetBytes(output, "tools.0.functionDeclarations.0.parametersJsonSchema.properties.mode.enum.1").String(); got != "b" {
		t.Fatalf("enum value = %q, want b. Output: %s", got, outputText)
	}
}
