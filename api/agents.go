package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var client = &http.Client{
	Timeout: 60 * time.Second, // Set a timeout for requests
}

type Agent struct{
    Name string `json:"name"`
    Description string `json:"description"`
    Provider *Provider `json:"provider"`
    Tools     []*Tool     `json:"tools"`

}

type Provider struct {
    BaseUrl string`json:"baseurl"`
    GenerateEndpoint string`json:"generateendpoint"`
    ChatEndpoint string`json:"chatendpoint"`
    ApiKey string `json:"apiKey"`
}

func NewProvider(baseurl,generate,chat string) *Provider{
    return &Provider{
        BaseUrl: baseurl,
        GenerateEndpoint: generate,
        ChatEndpoint: chat,
    }
}

func (provider *Provider) Generate(agent string, prompt string, stream bool) (*http.Response, error) {
	url := provider.BaseUrl + provider.GenerateEndpoint

	// Construct JSON payload
	payload := map[string]interface{}{
		"model":  agent,
		"prompt": prompt,
		"stream": stream,
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal generate request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	return resp, nil
}

// Chat function to interact with the model
func (provider *Provider) Chat(agent *Agent, chat *Chat,stream bool) (*http.Response, error) {
	url := provider.BaseUrl + provider.ChatEndpoint

	// Construct JSON payload
	payload := map[string]interface{}{
		"model":    agent.Name,
		"messages": chat.Messages,
		"stream":   stream,
        "tools":  agent.Tools,
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
    fmt.Println(string(jsonData))
	if err != nil {
		return nil, fmt.Errorf("unable to marshal chat messages: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	return resp, nil
}

type Chat struct{ 
    Agent       *Agent  `json:"Agent"`
    Messages []*Message `json:"messages"`
}

func NewChat(agent *Agent)*Chat{
    return &Chat{
        Agent: agent,
        Messages:make([]*Message, 0), 
    }
}


type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
	ToolCalls      []ToolCall   `json:"tool_calls,omitempty"`
}

type ChatResponse struct {
	Model          string       `json:"model"`
	CreatedAt      time.Time    `json:"created_at"`
	Response       string       `json:"response,omitempty"`
    Message       Message       `json:"message,omitempty"`
	Done           bool         `json:"done"`
	TotalDuration  int64        `json:"total_duration"`
	LoadDuration   int64        `json:"load_duration"`
	PromptEvalCount int         `json:"prompt_eval_count"`
	PromptEvalDuration int64    `json:"prompt_eval_duration"`
	EvalCount      int          `json:"eval_count"`
	EvalDuration   int64        `json:"eval_duration"`
}

type ToolCall struct {
	Function Function `json:"function"`
}
type Function struct {
	Name      string    `json:"name"`
	Arguments Arguments `json:"arguments"`
    FunctionCall func(arguments Arguments)map[string]interface{}
}

type Arguments struct {
	Format   string `json:"format"`
	Location string `json:"location"`
}// Tool represents a function tool that can be registered.

type Tool struct {
	Type     string      `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction defines the structure of a tool function.
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  ToolParameters         `json:"parameters"`
}

// ToolParameters defines the parameters required by a tool function.
type ToolParameters struct {
	Type       string                            `json:"type"`
    Properties map[string]*ToolParameterProperty 
	Required   []string                          `json:"required"`
}

func NewToolParameters(Type string)*ToolParameters{
    return &ToolParameters{
        Type: Type,
        Properties:make(map[string]*ToolParameterProperty), 
        Required: make([]string, 0),
    }
}


func(toolParameters *ToolParameters)AddProperty(propertyName, Type, desciption string, enum []string, required bool){
    toolParameters.Properties[propertyName] = NewToolParameterProperty(Type, desciption, enum, required)    
    if required{
        toolParameters.Required = append(toolParameters.Required,propertyName)
    }
}


// ToolParameterProperty defines a single parameter for a tool function.
type ToolParameterProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
	Required     bool `json:"required,omitempty"`

}


func NewToolParameterProperty(Type, desciption string, enum []string, required bool)*ToolParameterProperty{
    return &ToolParameterProperty{
        Type: Type,
        Description: desciption,
        Enum: enum,
        Required: required,
    }
}  




// ToolRegistry manages registered tools.
type ToolRegistry struct {
	Tools map[string]*Tool
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{Tools: make(map[string]*Tool)}
}

// RegisterTool adds a new tool to the registry.
func (tr *ToolRegistry) RegisterTool(tool *Tool) {
	tr.Tools[tool.Function.Name] = tool
}

// GetTool retrieves a registered tool by name.
func (tr *ToolRegistry) GetTool(name string) (*Tool, bool) {
	tool, exists := tr.Tools[name]
	return tool, exists
}

// GetTools returns the list of registered tools.
func (tr *ToolRegistry) GetTools() []*Tool {
	tools := make([]*Tool, 0, len(tr.Tools))
	for _, tool := range tr.Tools {
		tools = append(tools, tool)
	}
	return tools
}













