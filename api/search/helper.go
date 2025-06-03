package search

import (
	"fmt"
	"github.com/EdersenC/goAgent"
	"os"
	"strings"
)

type ExtractionResult struct {
	Citations []Citation `json:"citations"`
	Summary   string     `json:"summary"`
}

func (ex ExtractionResult) JoinCitations() string {
	var sb strings.Builder
	for _, citation := range ex.Citations {
		sb.WriteString(fmt.Sprintf("Content: %s\nURL: %s\nRelevance: %.2f\n\n", citation.Content, citation.URL, citation.Relevance))
	}
	return sb.String()
}

func newExtractionResult(citations []Citation, summary string) *ExtractionResult {
	return &ExtractionResult{
		Citations: citations,
		Summary:   summary,
	}
}

type Citation struct {
	Content   string  `json:"content,extracted_content"`
	URL       string  `json:"url,source"`
	Relevance float64 `json:"relevance"` //Todo Use embedding to calculate relevance
}

// tries to summarise a single chunk; retries once when BindToolResult fails
func summariseChunk(chunk, instructions string, maxContext int,
	chat *goAgent.Chat) (string, error) {

	// shrink chunk if it still busts the context window
	if goAgent.Tokenize(chat.Agent.SystemPrompt+chunk) > maxContext {
		chunk = strings.Join(goAgent.ChunkByTokens(chunk, maxContext), "\n")
	}

	const maxAttempts = 2
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		prompt := buildPrompt(instructions, chunk)
		fmt.Println("Prompt TokenSize", goAgent.Tokenize(prompt))
		response, err := chat.SendUserMessage(prompt, false)
		if err != nil {
			chat.ClearConversation()
			return "", err
		}

		var ext ExtractionResult
		if bindErr := response.Message.BindToolResult(
			searchExtraction.Function.Name, &ext); bindErr != nil {

			chat.ClearConversation()
			if attempt < maxAttempts {
				continue // retry once
			}
			_, err = fmt.Fprintf(os.Stderr, "Failed to bind tool result: %v\n", bindErr)
			if err != nil {
				return "", err
			}
			return "", bindErr
		}

		message := fmt.Sprintf("Summary: %s\n\nCitations:\n%s", ext.Summary, ext.JoinCitations())
		chat.ClearConversation()
		return message, nil
	}
	return "", fmt.Errorf("unreachable")
}

func ProcessChunks(chunks []string, chat *goAgent.Chat,
	instructions string, maxContext int) []string {

	var results []string
	for _, chunk := range chunks {
		msg, err := summariseChunk(chunk, instructions, maxContext, chat)
		if err != nil {
			fmt.Println("Chunk failed:", err)
			continue
		}
		results = append(results, msg)
	}
	return results
}

func ReviewExtraction(response map[string]interface{}, chat *goAgent.Chat) (map[string]interface{}, error) {
	arguments, ok := response["arguments"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("arguments not found in response")
	}

	citations, ok := arguments["citations"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("citations not found in arguments")
	}
	summary, ok := arguments["summary"].(string)
	if !ok {
		return nil, fmt.Errorf("summary not found in arguments")
	}
	// Build the result map
	result := map[string]interface{}{
		"citations": citations,
		"summary":   summary,
	}

	fmt.Println("Summary:", summary)
	fmt.Println("Citations:", citations)
	return result, nil
}

func buildPrompt(instr, content string) string {
	return fmt.Sprintf("%s\n\nExtract key information:\n\n%s", instr, content)
}
