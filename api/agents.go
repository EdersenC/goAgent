package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/EdersenC/goAgent/api/tools"
	"net/http"
	"os"
	"time"
)

var client = &http.Client{
	Timeout: 60 * time.Second, // Set a timeout for requests
}
var handoff = NewTool(
	"Function",
	"Handoff",
	"Routes a prompt to the most appropriate agent based on its content and intent.",
	tools.HandOff,
)

type Agent struct {
	Name        string    `json:"name"`
	Model       string    `json:"model"`
	Description string    `json:"description"`
	Provider    *Provider `json:"provider"`
	Tools       []*Tool   `json:"tools,omitempty"`
}

type Provider struct {
	BaseUrl          string `json:"baseurl"`
	GenerateEndpoint string `json:"generateendpoint"`
	ChatEndpoint     string `json:"chatendpoint"`
	ApiKey           string `json:"apiKey"`
}

func LoadAgents(file *os.File, agents *map[string]*Agent) error {
	bind := BindJSON(file, agents)
	if bind != nil {
		return fmt.Errorf("failed to load agents from %s", file.Name())
	}
	return nil
}

// Chat function to interact with the model
func (agent *Agent) Chat(chat *Chat, stream bool) (*http.Response, error) {
	url := agent.Provider.BaseUrl + agent.Provider.ChatEndpoint

	// Construct JSON payload
	payload := map[string]interface{}{
		"model":    agent.Model,
		"messages": chat.Messages,
		"stream":   stream,
		"tools":    agent.Tools,
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
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

	chatResponse, err := DecodeChatResponse(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Message asked:", chat.Messages[len(chat.Messages)-1].Content)
	chat.AddMessage(chatResponse.Message.Role, chatResponse.Message.Content)
	chat.RunTools(&chatResponse.Message)
	chatResponse.PrintResponse()
	return resp, nil
}

func (chat Chat) RunTools(message *Message) {
	if len(message.ToolCalls) > 0 {
		for _, function := range message.ToolCalls {
			toolCall, ok := function["function"].(map[string]interface{})
			toolName, ok := toolCall["name"].(string)
			if !ok {
				fmt.Println("Tool name not found in tool call")
				continue
			}
			tool, ok := chat.ToolRegistry.Tools[toolName]
			if !ok {
				fmt.Printf("Tool %s not found\n", toolName)
				continue
			}
			toolCall["caller"] = chat.Agent.Name
			tool.Function.FunctionCall(toolCall)
		}
	}
}

func (agent *Agent) AsTool(functionCall func(map[string]interface{}) map[string]interface{}) *Tool {
	tool := NewTool("agent", agent.Name, agent.Description, functionCall)
	tool.Function.Parameters.AddProperty(
		"message",
		"string",
		"The message to be sent to the agent",
		nil,
		true,
	)
	tool.Function.Parameters.AddProperty(
		"reason",
		"string",
		"You must specify reason for selecting this agent",
		nil,
		true,
	)
	return tool
}

func NewProvider(baseurl, generate, chat string) *Provider {
	return &Provider{
		BaseUrl:          baseurl,
		GenerateEndpoint: generate,
		ChatEndpoint:     chat,
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

// Chat represents a conversation with an agent.
type Chat struct {
	Agent        *Agent        `json:"Agent"`
	Messages     []*Message    `json:"messages"`
	ToolRegistry *ToolRegistry `json:"omitempty"`
}

func NewChat(agent *Agent, registry *ToolRegistry) *Chat {
	return &Chat{
		Agent:        agent,
		Messages:     make([]*Message, 0),
		ToolRegistry: registry,
	}
}
func (chat *Chat) AddMessage(role, content string) {
	localTime := time.Now()
	formattedTime := localTime.Format("Mon,2006-01-02 03:04:05 PM MST -0700")
	newContent := fmt.Sprintf("**This is the Current time and date:(%s)** **Prompt:** %s", formattedTime, content)
	message := NewMessage(role, newContent)
	message.Time = localTime
	chat.Messages = append(chat.Messages, message)
}

type Message struct {
	Role      string                   `json:"role"`
	Content   string                   `json:"content"`
	Images    []string                 `json:"images,omitempty"`
	ToolCalls []map[string]interface{} `json:"tool_calls,omitempty"`
	Time      time.Time                `json:"time"`
}

func NewMessage(role, content string) *Message {
	return &Message{
		Role:    role,
		Content: content,
		Images:  make([]string, 0),
	}
}
func (message *Message) AddImage(image string) {
	message.Images = append(message.Images, image)
}
func (message *Message) AddImages(images []string) {
	message.Images = append(message.Images, images...)
}

type ChatResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response,omitempty"`
	Message            Message   `json:"message,omitempty"`
	Done               bool      `json:"done"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int64     `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int64     `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function,omitempty"`
}

// ToolFunction defines the structure of a tool function.
type ToolFunction struct {
	Name         string                                              `json:"name"`
	Description  string                                              `json:"description"`
	Parameters   ToolParameters                                      `json:"parameters,omitempty"`
	FunctionCall func(map[string]interface{}) map[string]interface{} `json:"-"`
}

func NewTool(Type, name, description string, functionCall func(map[string]interface{}) map[string]interface{}) *Tool {
	return &Tool{
		Type: Type,
		Function: ToolFunction{
			Name:         name,
			Description:  description,
			FunctionCall: functionCall,
		},
	}
}

// ToolParameters defines the parameters required by a tool function.
type ToolParameters struct {
	Type       string                            `json:"type"`
	Properties map[string]*ToolParameterProperty `json:"properties,omitempty"`
	Required   []string                          `json:"required"`
}

func NewToolParameters(Type string) *ToolParameters {
	return &ToolParameters{
		Type:       Type,
		Properties: make(map[string]*ToolParameterProperty),
		Required:   make([]string, 0),
	}
}

func (toolParameters *ToolParameters) AddProperty(propertyName, Type, description string, enum []string, required bool) {
	if toolParameters.Properties == nil {
		toolParameters.Properties = make(map[string]*ToolParameterProperty)
	}
	toolParameters.Properties[propertyName] = NewToolParameterProperty(Type, description, enum, required)
	if required {
		toolParameters.Required = append(toolParameters.Required, propertyName)
	}
}

// ToolParameterProperty defines a single parameter for a tool function.
type ToolParameterProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
	Required    bool     `json:"required,omitempty"`
}

func NewToolParameterProperty(Type, description string, enum []string, required bool) *ToolParameterProperty {
	return &ToolParameterProperty{
		Type:        Type,
		Description: description,
		Enum:        enum,
		Required:    required,
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

// RegisterTool adds a single tool to the registry.
func (tr *ToolRegistry) RegisterTool(tool *Tool) {
	tr.Tools[tool.Function.Name] = tool
}

// RegisterTools adds one or more tools to the registry.
func (tr *ToolRegistry) RegisterTools(tools ...*Tool) {
	for _, tool := range tools {
		tr.Tools[tool.Function.Name] = tool
	}
}

// GetTools returns the list of registered tools.
func (tr *ToolRegistry) GetTools() []*Tool {
	Tools := make([]*Tool, 0, len(tr.Tools))
	for _, tool := range tr.Tools {
		Tools = append(Tools, tool)
	}
	return Tools
}
