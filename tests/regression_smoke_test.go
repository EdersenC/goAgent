package tests

import (
	"strings"
	"testing"

	goAgent "github.com/EdersenC/goAgent"
)

func TestDecodeChatResponseExtractsThinkingToolAndFinalContent(t *testing.T) {
	body := `{
		"model":"unit-test",
		"created_at":"2026-01-01T00:00:00Z",
		"done":true,
		"message":{
			"role":"assistant",
			"content":"<think>internal reasoning</think>Final answer<tool_call>{\"name\":\"search\",\"arguments\":{\"query\":\"weather\"}}</tool_call>"
		}
	}`

	resp, err := goAgent.DecodeChatResponse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("DecodeChatResponse returned error: %v", err)
	}

	if got, want := resp.Message.Thinking, "internal reasoning"; got != want {
		t.Fatalf("unexpected thinking: got %q want %q", got, want)
	}

	if got, want := resp.Message.Content, "Final answer"; got != want {
		t.Fatalf("unexpected cleaned content: got %q want %q", got, want)
	}

	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.Message.ToolCalls))
	}

	fn, ok := resp.Message.ToolCalls[0]["function"].(map[string]interface{})
	if !ok {
		t.Fatalf("tool call missing function payload: %#v", resp.Message.ToolCalls[0])
	}

	if got, ok := fn["name"].(string); !ok || got != "search" {
		t.Fatalf("unexpected tool name payload: %#v", fn["name"])
	}
}

func TestProviderGetChatURL(t *testing.T) {
	providerWithPort := &goAgent.Provider{
		BaseUrl:      "http://localhost",
		Port:         "11434",
		ChatEndpoint: "/api/chat",
	}
	if got, want := providerWithPort.GetChatUrl(), "http://localhost:11434/api/chat"; got != want {
		t.Fatalf("chat URL with port mismatch: got %q want %q", got, want)
	}

	providerWithoutPort := &goAgent.Provider{
		BaseUrl:      "https://api.example.com",
		ChatEndpoint: "/v1/chat",
	}
	if got, want := providerWithoutPort.GetChatUrl(), "https://api.example.com/v1/chat"; got != want {
		t.Fatalf("chat URL without port mismatch: got %q want %q", got, want)
	}
}
