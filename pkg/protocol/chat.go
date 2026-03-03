// Package protocol provides chat-related protocol types
package protocol

// ChatSendParams are the parameters for chat.send RPC
type ChatSendParams struct {
	// Message is the user message text
	Message string `json:"message"`
	// SessionID is the session identifier (optional, uses default if empty)
	SessionID string `json:"sessionId,omitempty"`
	// Model is the model override (optional)
	Model string `json:"model,omitempty"`
	// Stream enables streaming responses (default true)
	Stream *bool `json:"stream,omitempty"`
	// Tools enables tool calling (default true)
	Tools *bool `json:"tools,omitempty"`
}

// ChatSendPayload is the response payload for chat.send
type ChatSendPayload struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// MessageID is the message identifier
	MessageID string `json:"messageId"`
}

// ChatHistoryParams are the parameters for chat.history RPC
type ChatHistoryParams struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// Limit is the maximum number of messages to return
	Limit *int `json:"limit,omitempty"`
	// Before is a message ID to paginate before
	Before string `json:"before,omitempty"`
}

// ChatHistoryPayload is the response payload for chat.history
type ChatHistoryPayload struct {
	// Messages is the list of messages
	Messages []Message `json:"messages"`
	// HasMore indicates if there are more messages
	HasMore bool `json:"hasMore"`
}

// Message represents a single message in the history
type Message struct {
	// ID is the unique message identifier
	ID string `json:"id"`
	// Role is the message role (user, assistant, system, tool)
	Role string `json:"role"`
	// Content is the message content
	Content string `json:"content"`
	// Timestamp is the message timestamp (Unix seconds)
	Timestamp int64 `json:"timestamp"`
	// ToolCalls contains tool call data (for role == "tool")
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}

// ToolCall represents a tool call in a message
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

// ThinkingEventPayload is the payload for chat.thinking event
type ThinkingEventPayload struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// MessageID is the message identifier
	MessageID string `json:"messageId"`
}

// TextDeltaEventPayload is the payload for chat.textDelta event
type TextDeltaEventPayload struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// MessageID is the message identifier
	MessageID string `json:"messageId"`
	// Delta is the text delta (may be partial)
	Delta string `json:"delta"`
	// Done indicates if the message is complete
	Done bool `json:"done"`
}

// ToolStartEventPayload is the payload for chat.toolStart event
type ToolStartEventPayload struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// MessageID is the message identifier
	MessageID string `json:"messageId"`
	// ToolCallID is the tool call identifier
	ToolCallID string `json:"toolCallId"`
	// ToolName is the tool name
	ToolName string `json:"toolName"`
}

// ToolDeltaEventPayload is the payload for chat.toolDelta event
type ToolDeltaEventPayload struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// MessageID is the message identifier
	MessageID string `json:"messageId"`
	// ToolCallID is the tool call identifier
	ToolCallID string `json:"toolCallId"`
	// Delta is the result delta
	Delta string `json:"delta"`
	// Done indicates if the tool call is complete
	Done bool `json:"done"`
}

// ToolEndEventPayload is the payload for chat.toolEnd event
type ToolEndEventPayload struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// MessageID is the message identifier
	MessageID string `json:"messageId"`
	// ToolCallID is the tool call identifier
	ToolCallID string `json:"toolCallId"`
	// ToolName is the tool name
	ToolName string `json:"toolName"`
	// Success indicates if the tool call succeeded
	Success bool `json:"success"`
}

// DoneEventPayload is the payload for chat.done event
type DoneEventPayload struct {
	// SessionID is the session identifier
	SessionID string `json:"sessionId"`
	// MessageID is the message identifier
	MessageID string `json:"messageId"`
	// FinishReason is the reason for completion (stop, length, tool_calls)
	FinishReason string `json:"finishReason"`
	// TokensUsed is the number of tokens used
	TokensUsed int `json:"tokensUsed"`
}
