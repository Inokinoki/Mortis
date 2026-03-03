// Package provider provides LLM provider abstractions for Mortis
package provider

import (
	"context"
	"sync"
)

// Message represents a message in a conversation
type Message struct {
	// Role is the message role (system, user, assistant, tool)
	Role string `json:"role"`
	// Content is the message content
	Content string `json:"content"`
	// ToolCalls contains tool calls (for role == "tool")
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}

// ToolCall represents a tool/function call from the LLM
type ToolCall struct {
	// ID is the tool call identifier
	ID string `json:"id"`
	// Name is the tool name
	Name string `json:"name"`
	// Arguments are the tool arguments (JSON object)
	Arguments []byte `json:"arguments"`
	// Result is the tool result (empty if pending)
	Result string `json:"result,omitempty"`
}

// ToolResult is the result of a tool call
type ToolResult struct {
	// ID is the tool call identifier
	ID string `json:"id"`
	// Success indicates if the tool call succeeded
	Success bool `json:"success"`
	// Result is the tool result
	Result string `json:"result"`
}

// CompletionRequest is a request for completion (non-streaming)
type CompletionRequest struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// MessageID is the message identifier
	MessageID string `json:"messageId"`
	// Messages is the conversation history
	Messages []Message `json:"messages"`
	// System is the system prompt
	System string `json:"system,omitempty"`
	// Model is the model to use
	Model string `json:"model,omitempty"`
	// ToolsEnabled enables tool calling
	ToolsEnabled bool `json:"toolsEnabled"`
	// MaxTokens is the maximum tokens to generate
	MaxTokens int `json:"maxTokens,omitempty"`
	// Temperature is the sampling temperature
	Temperature float64 `json:"temperature,omitempty"`
	// TopP is the nucleus sampling parameter
	TopP float64 `json:"top_p,omitempty"`
}

// CompletionResponse is the response to a completion request
type CompletionResponse struct {
	// Content is the generated text
	Content string `json:"content"`
	// FinishReason is the reason for completion
	FinishReason string `json:"finishReason"`
	// TokensUsed is the number of tokens used
	TokensUsed int `json:"tokensUsed"`
	// ToolCalls contains tool calls requested by the LLM
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}

// StreamEvent is an event during streaming
type StreamEvent struct {
	// Type is the event type
	Type string `json:"type"`
	// Content is the text delta (for type == "text_delta")
	Content string `json:"content,omitempty"`
	// ToolCall is the tool call (for type == "tool_call")
	ToolCall *ToolCall `json:"toolCall,omitempty"`
	// ToolDelta is the tool result delta (for type == "tool_delta")
	ToolDelta *ToolDelta `json:"toolDelta,omitempty"`
	// FinishReason is the reason for completion (for type == "done")
	FinishReason string `json:"finishReason,omitempty"`
	// TokensUsed is the number of tokens used (for type == "done")
	TokensUsed int `json:"tokensUsed,omitempty"`
}

// ToolDelta represents a delta in tool result during streaming
type ToolDelta struct {
	// ID is the tool call identifier
	ID string `json:"id"`
	// Delta is the result delta
	Delta string `json:"delta"`
	// Done indicates if the tool call is complete
	Done bool `json:"done"`
}

// StreamEvent types
const (
	StreamEventTypeTextDelta = "text_delta"
	StreamEventTypeToolCall  = "tool_call"
	StreamEventTypeToolDelta = "tool_delta"
	StreamEventTypeDone      = "done"
)

// Info contains provider information
type Info struct {
	// ID is the provider identifier
	ID string `json:"id"`
	// Name is the provider name
	Name string `json:"name"`
	// Type is the provider type
	Type string `json:"type"`
	// Available indicates if the provider is available
	Available bool `json:"available"`
	// Models is the list of available models
	Models []ModelInfo `json:"models,omitempty"`
	// Features are the features supported by the provider
	Features []string `json:"features,omitempty"`
}

// ModelInfo contains model information
type ModelInfo struct {
	// ID is the model identifier
	ID string `json:"id"`
	// Name is the model name
	Name string `json:"name"`
	// ContextSize is the context window size
	ContextSize int `json:"contextSize,omitempty"`
	// Description is the model description
	Description string `json:"description,omitempty"`
}

// LLM is the interface that all LLM providers must implement
type LLM interface {
	// Info returns provider information
	Info(ctx context.Context) (*Info, error)

	// Complete generates a completion (non-streaming)
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

	// Stream generates a streaming completion
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, error)

	// Close closes any resources
	Close() error
}

// Registry manages provider registration
type Registry struct {
	providers       map[string]LLM
	defaultProvider string
	mu              sync.RWMutex
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]LLM),
	}
}

// Register registers a provider
func (r *Registry) Register(id string, provider LLM) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[id] = provider
}

// Unregister unregisters a provider
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, id)
}

// Get returns a provider by ID
func (r *Registry) Get(id string) (LLM, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

// List returns all providers
func (r *Registry) List() map[string]LLM {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]LLM)
	for k, v := range r.providers {
		result[k] = v
	}
	return result
}

// SetDefault sets the default provider
func (r *Registry) SetDefault(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultProvider = id
}

// GetDefault returns the default provider
func (r *Registry) GetDefault() (LLM, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.defaultProvider == "" {
		return nil, false
	}
	p, ok := r.providers[r.defaultProvider]
	return p, ok
}

// Features supported by providers
const (
	FeatureCompletion  = "completion"
	FeatureStreaming   = "streaming"
	FeatureToolCalling = "tool_calling"
	FeatureVision      = "vision"
	FeatureEmbeddings  = "embeddings"
)

// DefaultContextSize is the default context size for models without a specified size
const DefaultContextSize = 128000

// DefaultMaxTokens is the default maximum tokens to generate
const DefaultMaxTokens = 4096

// DefaultTemperature is the default sampling temperature
const DefaultTemperature = 0.7

// DefaultTopP is the default nucleus sampling parameter
const DefaultTopP = 1.0
