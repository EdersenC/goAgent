package goAgent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var client = &http.Client{
	Timeout: 10 * time.Minute, // Set a timeout for requests
}

var EmbeddingAgent *Agent
var SummaryAgent *Agent
var PlannerAgent *Agent

type Agent struct {
	Name         string        `json:"name"`
	Model        Model         `json:"model"`
	Description  string        `json:"description"`
	Provider     *Provider     `json:"provider"`
	Language     string        `json:"language,omitempty"`
	SystemPrompt string        `json:"systemPrompt,omitempty"`
	Tools        *ToolRegistry `json:"tools,omitempty"`
}

// Clone creates a deep copy of the Agent instance.
func (a *Agent) Clone() *Agent {
	agentCopy := &Agent{
		Name:        a.Name,
		Model:       a.Model,
		Description: a.Description,
	}
	if a.Provider != nil {
		agentCopy.Provider = &Provider{
			BaseUrl:           a.Provider.BaseUrl,
			Port:              a.Provider.Port,
			GenerateEndpoint:  a.Provider.GenerateEndpoint,
			ChatEndpoint:      a.Provider.ChatEndpoint,
			EmbeddingEndpoint: a.Provider.EmbeddingEndpoint,
			TokenizeEndpoint:  a.Provider.TokenizeEndpoint,
			ApiKey:            a.Provider.ApiKey,
		}
	}
	if a.Language != "" {
		agentCopy.Language = a.Language
	}
	if a.SystemPrompt != "" {
		agentCopy.SystemPrompt = a.SystemPrompt
	}
	if a.Tools != nil {
		agentCopy.Tools = a.Tools // Provide a shallow copy of the ToolRegistry
	} else {
		agentCopy.Tools = NewToolRegistry()
	}
	return agentCopy
}

// WithPort sets the port for the agent's provider.
func (a *Agent) WithPort(port string) *Agent {
	clone := a.Clone()
	if clone.Provider == nil {
		panic("Provider must be set before setting the port")
	}
	clone.Provider.Port = port
	return clone
}

// clearTools clears the agent's tools and returns the previous tools.
// It returns an empty slice if no tools were set.
func (a *Agent) ClearTools() *ToolRegistry {
	return a.Tools.Clear()
}

// RegisterTools sets the tools for the agent.
// It appends valid tools to the agent's tool list, ignoring nil or invalid tools.
func (a *Agent) RegisterTools(tools ...*Tool) {
	if a.Tools == nil {
		a.Tools = NewToolRegistry()
	}
	a.Tools.RegisterTools(tools...)
}

// GetTools retrieves the agent's tool registry.
func (a *Agent) GetTools() *ToolRegistry {
	if a.Tools == nil {
		a.Tools = NewToolRegistry()
	}
	return a.Tools
}

// GetToolMap retrieves the internal map of tools from the agent's tool registry.
func (a *Agent) GetToolMap() map[string]*Tool {
	if a.Tools == nil {
		a.Tools = NewToolRegistry()
	}
	return a.Tools.GetToolMap()
}

// SwapRegistry swaps the agent's tool registry with a new one.
func (a *Agent) SwapRegistry(registry *ToolRegistry) *ToolRegistry {
	if a.Tools == nil {
		a.Tools = NewToolRegistry()
	}
	return a.Tools.Swap(registry)
}

// GetToolsByName retrieves tools by their names from the agent's tool registry.
// It returns an error if no tools are found for the specified names.
func (a *Agent) GetToolsByName(name ...string) (*ToolRegistry, error) {
	if a.Tools == nil {
		a.Tools = NewToolRegistry()
	}
	registry := a.Tools.GetToolsByName(name...)
	if len(registry.Tools) == 0 {
		return nil, fmt.Errorf("no tools found for names: %v", name)
	}
	return registry, nil
}

type AgentMesh struct {
	Agents map[string]*Agent `json:"agents"`
}

type Model struct {
	Name          string `json:"name"`
	ContextWindow int    `json:"contextWindow"`
	Reasoning     bool   `json:"reasoning,omitempty"`
}

type Provider struct {
	BaseUrl           string `json:"baseurl"`
	Port              string `json:"port,omitempty"`
	GenerateEndpoint  string `json:"generateEndpoint"`
	ChatEndpoint      string `json:"chatEndpoint"`
	EmbeddingEndpoint string `json:"embeddingEndpoint"`
	TokenizeEndpoint  string `json:"tokenizeEndpoint,omitempty"`
	ApiKey            string `json:"apiKey"`
}

// GetChatUrl constructs the chat URL for the provider.
func (p *Provider) GetChatUrl() string {
	if p.Port != "" {
		return fmt.Sprintf("%s:%s%s", p.BaseUrl, p.Port, p.ChatEndpoint)
	}
	return fmt.Sprintf("%s%s", p.BaseUrl, p.ChatEndpoint)
}

// getGenerateUrl constructs the generate URL for the provider.
func (p *Provider) getGenerateUrl() string {
	if p.Port != "" {
		return fmt.Sprintf("%s:%s%s", p.BaseUrl, p.Port, p.GenerateEndpoint)
	}
	return fmt.Sprintf("%s%s", p.BaseUrl, p.GenerateEndpoint)
}

// getEmbeddingUrl constructs the embedding URL for the provider.
func (p *Provider) getEmbeddingUrl() string {
	if p.Port != "" {
		return fmt.Sprintf("%s:%s%s", p.BaseUrl, p.Port, p.EmbeddingEndpoint)
	}
	return fmt.Sprintf("%s%s", p.BaseUrl, p.EmbeddingEndpoint)
}

// ProvideOllama creates a new Provider instance for Ollama with default settings.
func ProvideOllama() *Provider {
	return &Provider{
		BaseUrl:           "http://localhost",
		Port:              "11434",
		GenerateEndpoint:  "/api/generate",
		ChatEndpoint:      "/api/chat",
		EmbeddingEndpoint: "/api/embed",
	}
}

func NewProvider(baseurl, generate, chat string) *Provider {
	return &Provider{
		BaseUrl:          baseurl,
		GenerateEndpoint: generate,
		ChatEndpoint:     chat,
	}
}

// ProvideOllamaWithPort creates a new Provider instance for Ollama with a specified port.
func LoadAgents(file *os.File, agents *map[string]*Agent) error {
	bind := BindJSON(file, agents)
	if bind != nil {
		return fmt.Errorf("failed to load agents from %s", file.Name())
	}
	return nil
}

// ContextPortion ContextPortionFloat ContextPortion calculates the portion of the context window based on the given percentage.
// It returns the number of tokens that correspond to the specified percentage of the agent's context window.
func (a *Agent) ContextPortion(percentage float64) int {
	if a.Model.ContextWindow <= 0 || percentage <= 0 {
		return 0
	}
	return int(float64(a.Model.ContextWindow) * (percentage / 100))
}

// ContextPortionFloat calculates the portion of the context window as a float64 value based on the given percentage.
func (a *Agent) ContextPortionFloat(percentage float64) float64 {
	if a.Model.ContextWindow <= 0 || percentage <= 0 {
		return 0.0
	}
	return float64(a.Model.ContextWindow) * (percentage / 100)
}

// embedChunked calculates the portion of the context window for embedding based on the given percentage.
func (a *Agent) embedChunked(percentage float64) float64 {
	if a.Model.ContextWindow <= 0 || percentage <= 0 {
		return 0.0
	}
	return float64(a.Model.ContextWindow) * (percentage / 100)
}

type EmbeddedContent struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Embedding []float64 `json:"embedding"`
}

// Embed embeds the provided content by chunking it into smaller pieces and embedding each chunk.
func (a *Agent) Embed(content string) ([]*EmbeddedContent, error) {
	embeddingContents := make([]*EmbeddedContent, 0)
	for _, chunk := range ChunkByTokens(content, a.ContextPortion(100)) {
		embeddedContent, err := a.EmbedChunk(chunk)
		if err != nil {
			return nil, fmt.Errorf("error embedding chunk: %w", err)
		}
		if embeddedContent != nil {
			embeddingContents = append(embeddingContents, embeddedContent)
		}
	}

	return embeddingContents, nil
}

// EmbedChunk embeds a single chunk of content and returns the embedded content.
func (a *Agent) EmbedChunk(content string) (*EmbeddedContent, error) {
	url := a.Provider.getEmbeddingUrl()

	payload := map[string]interface{}{
		"model":  a.Model.Name,
		"prompt": content,
		"options": map[string]interface{}{
			"num_ctx": a.Model.ContextWindow,
		},
	}

	jsonData, err := marshalPayload(payload)
	if err != nil {
		return nil, err
	}

	req, err := createPostRequest(url, jsonData)
	if err != nil {
		return nil, err
	}

	body, err := doRequest(req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding embedding: %w", err)
	}
	embeddingContents := &EmbeddedContent{
		ID:        fmt.Sprintf("%s-%d", a.Model.Name, time.Now().UnixNano()),
		Content:   content, // Assuming content is a single string
		Embedding: result.Embedding,
	}

	return embeddingContents, nil
}

// AsTool converts the agent into a tool that can be used in a chat context.
func (a *Agent) AsTool(functionCall func(map[string]interface{}, *Chat) (map[string]interface{}, error)) *Tool {
	tool := NewTool("agent", a.Name, a.Description, functionCall)
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

// SendMessage sends a message to the agent and returns the response.
// It constructs the request payload, sends it to the agent's chat URL, and decodes the response.
func (c *Chat) SendMessage(role, content string, stream bool) (*ChatResponse, error) {
	url := c.Agent.Provider.GetChatUrl()

	c.AddMessage(role, content)
	payload := map[string]interface{}{
		"model":      c.Agent.Model.Name,
		"messages":   c.Messages,
		"stream":     stream,
		"tools":      c.Agent.Tools.GetTools(),
		"keep_alive": -1,
		"options": map[string]interface{}{
			"num_ctx": c.Agent.Model.ContextWindow,
		},
	}

	jsonData, err := marshalPayload(payload)
	if err != nil {
		return nil, err
	}

	req, err := createPostRequest(url, jsonData)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	chatResponse, err := DecodeChatResponse(resp.Body)
	if err != nil {
		fmt.Println("decode error:", err)
		return nil, err
	}

	c.AddMessage(chatResponse.Message.Role, chatResponse.Message.Thinking+chatResponse.Message.Content)
	c.RunTools(&chatResponse.Message)
	return chatResponse, nil
}

// RunTools executes the tools specified in the message's tool calls.
// It iterates over each tool call, retrieves the corresponding tool from the registry,
func (c *Chat) RunTools(message *Message) {
	if len(message.ToolCalls) > 0 {
		for i, _ := range message.ToolCalls {
			toolCall, ok := message.ToolCalls[i]["function"].(map[string]interface{})
			toolName, ok := toolCall["name"].(string)
			if !ok {
				fmt.Println("Tool name not found in tool call")
				continue
			}
			tool, ok := c.ToolRegistry.Tools[toolName]
			if !ok {
				fmt.Printf("Tool %s not found\n", toolName)
				continue
			}
			toolCall["caller"] = c.Agent.Name
			toolCall["prompt"] = c.Messages[len(c.Messages)-2].Content

			results, err := tool.Call(toolCall, c)
			if err != nil {
				fmt.Printf("Error calling%s:%s\n", toolName, err)
				continue
			}
			toolCall["result"] = results
		}
	}
}

// Chat represents a conversation with an agent.
type Chat struct {
	Agent        *Agent        `json:"Agent"`
	Messages     []*Message    `json:"messages"`
	ToolRegistry *ToolRegistry `json:"omitempty"`
}

// NewChat creates a new Chat instance with the specified agent and tool registry.
func NewChat(agent *Agent, registry *ToolRegistry) *Chat {
	return &Chat{
		Agent:        agent,
		Messages:     make([]*Message, 0),
		ToolRegistry: registry,
	}
}

// clear clears the chat messages without resetting the system prompt.
func (c *Chat) Clear() {
	c.Messages = make([]*Message, 0)
}

// ClearConversation clears the chat messages and resets the conversation with the agent's system prompt.
func (c *Chat) ClearConversation() {
	c.Messages = make([]*Message, 0)
	c.Messages = append(c.Messages, NewMessage("system", c.Agent.SystemPrompt))
}

// Swap swaps the messages and tools of the current chat with another chat.
// If the other chat is nil, it returns the current chat unchanged.
func (c *Chat) Swap(chat *Chat) *Chat {
	if chat == nil {
		return c
	}
	if c.Agent == nil {
		c.Agent = chat.Agent
	}
	if c.ToolRegistry == nil {
		c.ToolRegistry = chat.ToolRegistry
	}
	oldMessages := c.Messages
	c.Messages = chat.Messages
	chat.Messages = oldMessages
	return c
}

// AddMessage adds a new message to the chat.
func (c *Chat) AddMessage(role, content string) {
	localTime := time.Now()
	formattedTime := localTime.Format("Mon,2006-01-02 03:04:05 PM MST -0700")
	//todo make AddMessage more flexible like swap prompt for different roles
	newContent := content
	if role == "user" {
		newContent = fmt.Sprintf(
			"\n**The User's Current time and date is:** %s\n**The User Speaks:**\n%s\n\n%s\n",
			formattedTime,
			c.Agent.Language,
			content,
		)

	}
	message := NewMessage(role, newContent)
	message.Time = localTime
	c.Messages = append(c.Messages, message)
}

// SendUserMessage sends a user message to the agent and returns the response.
func (c *Chat) SendUserMessage(content string, stream bool) (*ChatResponse, error) {
	content = "**User Prompt**:\n " + content
	return c.SendMessage("user", content, stream)
}

// SendAssistantMessage sends an assistant message to the agent and returns the response.
func (c *Chat) SendAssistantMessage(content string, stream bool) (*ChatResponse, error) {
	content = "**Assistant Response**:\n " + content
	return c.SendMessage("assistant", content, stream)
}

// SendSystemMessage sends a system message to the agent and returns the response.
func (c *Chat) SendSystemMessage(content string, stream bool) (*ChatResponse, error) {
	content = "**System Message**:\n " + content
	return c.SendMessage("system", content, stream)
}

type Message struct {
	Role      string                   `json:"role"`
	Content   string                   `json:"content"`
	Thinking  string                   `json:"thinking"`
	Raw       string                   `json:"-"`
	Images    []string                 `json:"images,omitempty"`
	ToolCalls []map[string]interface{} `json:"tool_calls,omitempty"`
	Time      time.Time                `json:"time"`
}

// NewMessage creates a new Message instance with the specified role and content.
func NewMessage(role, content string) *Message {
	return &Message{
		Role:    role,
		Content: content,
		Images:  make([]string, 0),
	}
}

// BindToolResult binds a tool result to the provided key in the message.
// It searches for the tool call with the specified key and unmarshals the result into the provided variable.
func (m *Message) BindToolResult(key string, v interface{}) error {
	if m.ToolCalls == nil || len(m.ToolCalls) == 0 {
		return fmt.Errorf("no tool calls found in message")
	}

	if key == "" {
		return fmt.Errorf("tool call key cannot be empty")
	}

	if v == nil {
		return fmt.Errorf("bind target cannot be nil")
	}

	for _, toolCall := range m.ToolCalls {
		toolCall = toolCall["function"].(map[string]interface{})
		if name, ok := toolCall["name"].(string); ok && name == key {
			rawResult, ok := toolCall["result"]
			if !ok || rawResult == nil {
				return fmt.Errorf("tool call %s has no result", key)
			}

			jsonBytes, err := json.Marshal(rawResult)
			if err != nil {
				return fmt.Errorf("failed to marshal tool result: %w", err)
			}

			err = json.Unmarshal(jsonBytes, v)
			if err != nil {
				return fmt.Errorf("failed to unmarshal tool result: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("tool call %s not found in message", key)
}

// BindToolResults binds multiple tool results to the provided key in the message.
// It returns a slice of bound results or an error if any binding fails.
func (m *Message) BindToolResults(key string, v ...interface{}) ([]any, error) {
	var bindings = make([]any, 0)
	for _, value := range v {
		err := m.BindToolResult(key, value)
		if err != nil {
			fmt.Println("Error binding tool result:", err)
			continue
		}
		bindings = append(bindings, value)
	}

	return bindings, nil
}

// AddImages adds one or more images to the message.
func (m *Message) AddImages(images ...string) {
	m.Images = append(m.Images, images...)
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

// NewTool creates a new Tool instance with the specified type, name, description, and function call.
func NewTool(Type, name, description string, functionCall func(map[string]interface{}, *Chat) (map[string]interface{}, error)) *Tool {
	return &Tool{
		Type: Type,
		Function: ToolFunction{
			Name:         name,
			Description:  description,
			FunctionCall: functionCall,
		},
	}
}

// AddConstraints adds one or more constraints to the tool function.
func (t *Tool) AddConstraints(constraints ...string) {
	if t.Function.Constraints == nil {
		t.Function.Constraints = make([]string, 0)
	}
	for _, constraint := range constraints {
		if constraint != "" {
			t.Function.Constraints = append(t.Function.Constraints, constraint)
		}
	}
}

// AddExamples adds one or more examples to the tool function.
func (t *Tool) AddExamples(examples ...string) {
	if t.Function.Examples == nil {
		t.Function.Examples = make([]string, 0)
	}
	for _, example := range examples {
		if example != "" {
			t.Function.Examples = append(t.Function.Examples, example)
		}
	}
}

// Clone creates a deep copy of the Tool instance.
func (t *Tool) Clone() *Tool {
	toolCopy := &Tool{
		Type: t.Type,
		Function: ToolFunction{
			Name:         t.Function.Name,
			Description:  t.Function.Description,
			Examples:     make([]string, len(t.Function.Examples)),
			Constraints:  make([]string, len(t.Function.Constraints)),
			Parameters:   t.Function.Parameters,
			FunctionCall: t.Function.FunctionCall,
		},
	}
	copy(toolCopy.Function.Examples, t.Function.Examples)
	copy(toolCopy.Function.Constraints, t.Function.Constraints)
	return toolCopy
}

// getFunctionCall retrieves the function call associated with the tool.
func (t *Tool) getFunctionCall() (func(map[string]interface{}, *Chat) (map[string]interface{}, error), error) {
	if t.Function.FunctionCall == nil {
		return nil, fmt.Errorf("tool %s has no function call defined", t.Function.Name)
	}
	return t.Function.FunctionCall, nil
}

// Call executes the tool's function call with the provided arguments and chat context.
func (t *Tool) Call(arguments map[string]interface{}, chat *Chat) (map[string]interface{}, error) {
	functionCall, err := t.getFunctionCall()
	if err != nil {
		return nil, err
	}
	results, err := functionCall(arguments, chat)
	if err != nil {
		return nil, fmt.Errorf("error calling tool %s: %w", t.Function.Name, err)
	}
	return results, nil
}

// AsPrompt formats the tool's function description, examples, and constraints into a prompt string.
func (t *Tool) AsPrompt(maxExamples int) string {
	examples := ""
	if maxExamples < 0 {
		for i, example := range t.Function.Examples {
			examples += fmt.Sprintf("\nExample %d: %s\n", i+1, example)
		}
	} else {
		for i, example := range t.Function.Examples {
			if i >= maxExamples {
				break
			}
			examples += fmt.Sprintf("Example %d: %s", i+1, example)
		}
	}
	constraints := ""
	for i, constraint := range t.Function.Constraints {
		constraints += fmt.Sprintf("\n**%d**- %s", i+1, constraint)
	}
	return fmt.Sprintf("\nDescription: %s\nExamples: %s\nConstraints: %s",
		t.Function.Description,
		examples,
		constraints,
	)
}

// ToolFunction defines the structure of a tool function.
type ToolFunction struct {
	Name         string                                                              `json:"name"`
	Description  string                                                              `json:"description"`
	Examples     []string                                                            `json:"examples"` //Todo change to map[string]string for more flexibility
	Constraints  []string                                                            `json:"constraints"`
	Parameters   ToolParameters                                                      `json:"parameters,omitempty"`
	FunctionCall func(map[string]interface{}, *Chat) (map[string]interface{}, error) `json:"-"`
}

// ToolParameters defines the parameters required by a tool function.
type ToolParameters struct {
	Type       string                            `json:"type"`
	Properties map[string]*ToolParameterProperty `json:"properties,omitempty"`
	Required   []string                          `json:"required"`
}

// ToolParameterProperty defines a single parameter for a tool function.
type ToolParameterProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
	Required    bool     `json:"required,omitempty"`
}

// NewToolParameters creates a new ToolParameters instance with the specified type.
func NewToolParameters(Type string) *ToolParameters {
	return &ToolParameters{
		Type:       Type,
		Properties: make(map[string]*ToolParameterProperty),
		Required:   make([]string, 0),
	}
}

// AddProperty adds a new property to the tool parameters.
func (toolParameters *ToolParameters) AddProperty(propertyName, Type, description string, enum []string, required bool) {
	if toolParameters.Properties == nil {
		toolParameters.Properties = make(map[string]*ToolParameterProperty)
	}
	toolParameters.Properties[propertyName] = NewToolParameterProperty(Type, description, enum, required)
	if required {
		toolParameters.Required = append(toolParameters.Required, propertyName)
	}
}

// NewToolParameterProperty creates a new ToolParameterProperty instance with the specified type, description, enum values, and required status.
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
func NewToolRegistry(tool ...*Tool) *ToolRegistry {
	if len(tool) > 0 {
		registry := &ToolRegistry{Tools: make(map[string]*Tool)}
		for _, t := range tool {
			registry.RegisterTool(t)
		}
		return registry
	}
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

// Swap replaces the current tool registry with a new one and returns the previous tools.
func (tr *ToolRegistry) Swap(registry *ToolRegistry) *ToolRegistry {
	if registry == nil {
		return tr
	}
	if tr.Tools == nil {
		tr.Tools = make(map[string]*Tool)
	}
	oldRegistry := tr.GetTools()
	tr.Tools = registry.GetToolMap()
	return NewToolRegistry(oldRegistry...)
}

// GetToolMap returns the internal map of tools.
func (tr *ToolRegistry) GetToolMap() map[string]*Tool {
	if tr.Tools == nil {
		tr.Tools = make(map[string]*Tool)
	}
	return tr.Tools
}

// Clear removes all tools from the registry and returns a new ToolRegistry with the previously registered tools.
// It does not delete the tools but clears the registry's internal map.
func (tr *ToolRegistry) Clear() *ToolRegistry {
	tools := tr.GetTools()
	tr.Tools = make(map[string]*Tool)
	return NewToolRegistry(tools...)
}

// GetToolByName retrieves a tool by its name from the registry.
// It returns an error if the tool is not found.
func (tr *ToolRegistry) GetToolsByName(name ...string) *ToolRegistry {
	var registry = NewToolRegistry()
	if tr.Tools == nil {
		tr.Tools = make(map[string]*Tool)
		return registry
	}
	for _, n := range name {
		if tool, ok := tr.Tools[n]; ok {
			registry.RegisterTool(tool)
		}
	}
	return registry
}
