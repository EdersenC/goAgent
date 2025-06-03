package goAgent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func DecodeChatResponse(body io.Reader) (*ChatResponse, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	//bodyStr := string(bodyBytes)

	var response ChatResponse
	if err = json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	response.Message.Raw = response.Message.Content
	response.Message.ToolCalls = append(response.Message.ToolCalls, response.ExtractToolCalls()...)
	response.Message.Thinking = response.ExtractThinking()
	response.Message.Content = response.ExtractFinalContent()
	return &response, nil
}

// Returns parsed ToolCalls from the message content (no side effects).
func (cr *ChatResponse) ExtractToolCalls() []map[string]interface{} {
	re := regexp.MustCompile(`(?s)<tool_call>(.*?)</tool_call>`)
	matches := re.FindAllStringSubmatch(cr.Message.Content, -1)

	var toolCalls []map[string]interface{}
	for _, match := range matches {
		toolCallStr := strings.TrimSpace(match[1])
		var toolCall map[string]interface{}
		if err := json.Unmarshal([]byte(toolCallStr), &toolCall); err == nil {
			toolCalls = append(toolCalls, map[string]interface{}{"function": toolCall})
		} else {
			fmt.Println("Failed to parse tool_call JSON:", err)
		}
	}
	return toolCalls
}

// Returns the <think> content if found (no side effects).
func (cr *ChatResponse) ExtractThinking() string {
	re := regexp.MustCompile(`(?s)<think>\s*(.*?)\s*</think>`)
	matches := re.FindStringSubmatch(cr.Message.Content)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

var (
	// Compiled regexes to strip think/tool_call blocks
	reThink = regexp.MustCompile(`(?s)<think>.*?</think>`)
	reTool  = regexp.MustCompile(`(?s)<tool_call>.*?</tool_call>`)
)

// Returns cleaned message content with <think> and <tool_call> blocks removed (no side effects).
func (cr *ChatResponse) ExtractFinalContent() string {
	content := cr.Message.Content
	content = reThink.ReplaceAllString(content, "")
	content = reTool.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

func (cr *ChatResponse) PrintFullResponse() {
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
			fmt.Printf("Arguments: %+v\n", toolCall)
		}
	}
}

func (cr *ChatResponse) PrintThoughts() {
	fmt.Println("=====================\nThoughts:\n", cr.Message.Thinking)
}

func (cr *ChatResponse) PrintContent() {
	fmt.Println("=====================\nContents:\n", cr.Message.Content)
}

func marshalPayload(payload any) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal payload: %w", err)
	}
	return jsonData, nil
}

func createPostRequest(url string, jsonData []byte) (*http.Request, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func doRequest(req *http.Request) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}
	return body, nil
}

// Tokenize estimates the number of tokens in a prompt based on word count and character count.
// This is a heuristic approach and may not be accurate for all tokenization methods.
func Tokenize(prompt string) int {
	wordCount := len(strings.Fields(prompt))
	charCount := len(prompt)
	return wordCount + (charCount / 5) //Based word count + fudge for punctuation/symbols
}

// ChunkByTokens splits text into pieces that never exceed limit tokens.
func ChunkByTokens(text string, limit int) []string {
	var chunks []string
	var buf strings.Builder

	for _, line := range strings.Split(text, "\n") {
		if Tokenize(buf.String()+line) > limit {
			chunks = append(chunks, buf.String())
			buf.Reset()
		}
		buf.WriteString(line + "\n")
	}
	if buf.Len() > 0 {
		chunks = append(chunks, buf.String())
	}
	return chunks
}

func InitTool(tool *Tool, fileName string, function func(map[string]interface{}, *Chat) (map[string]interface{}, error)) {
	toolJson, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = LoadTool(toolJson, tool)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if function != nil {
		tool.Function.FunctionCall = function
	}
}

func LoadTool(file *os.File, tool *Tool) error {
	bind := BindJSON(file, tool)
	if bind != nil {
		return fmt.Errorf("failed to load tools from %s", file.Name())
	}
	return nil
}
