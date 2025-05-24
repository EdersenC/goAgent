package api

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

func DecodeChatResponse(body io.Reader) (*ChatResponse, error) {
	// Read the entire response body
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert body to string and print it
	bodyString := string(bodyBytes)
	fmt.Println("\n\nResponse Body:", bodyString)

	// Decode the JSON response into ChatResponse struct
	var response ChatResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return &response, nil
}

func (cr *ChatResponse) PrintResponse() {
	fmt.Printf("Model: %s\n", cr.Model)
	fmt.Printf("Created At: %s\n", cr.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Message: %s\n", cr.Response)
	fmt.Printf("Done: %t\n", cr.Done)
	fmt.Printf("Total Duration: %d ns\n", cr.TotalDuration)
	fmt.Printf("Load Duration: %d ns\n", cr.LoadDuration)
	fmt.Printf("Prompt Eval Count: %d\n", cr.PromptEvalCount)
	fmt.Printf("Prompt Eval Duration: %d ns\n", cr.PromptEvalDuration)
	fmt.Printf("Eval Count: %d\n", cr.EvalCount)
	fmt.Printf("Eval Duration: %d ns\n", cr.EvalDuration)
	if len(cr.Message.ToolCalls) > 0 {
		fmt.Println("Tool Calls:")
		for _, toolCall := range cr.Message.ToolCalls {
			fmt.Printf("   Arguments: %+v\n", toolCall)
		}
	}
}
