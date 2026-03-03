// Package protocol defines the WebSocket/RPC protocol types for Mortis.
// Protocol version 3. Compatible with Moltis protocol.
package protocol

const (
	// ProtocolVersion is the current protocol version
	ProtocolVersion = 3

	// MaxPayloadBytes is the maximum frame payload size (512 KB)
	MaxPayloadBytes = 524 * 1024

	// MaxBufferedBytes is the maximum buffered message size (1.5 MB)
	MaxBufferedBytes = 1572 * 1024

	// TickIntervalMs is the tick interval for heartbeats (30s)
	TickIntervalMs = 30000

	// HandshakeTimeoutMs is the timeout for WebSocket handshake (10s)
	HandshakeTimeoutMs = 10000

	// DedupeTTLMs is the time-to-live for deduplication cache (5 min)
	DedupeTTLMs = 300000

	// DedupeMaxEntries is the maximum number of deduplication entries
	DedupeMaxEntries = 1000
)

// Error codes
const (
	ErrorCodeNotLinked       = "NOT_LINKED"
	ErrorCodeNotPaired       = "NOT_PAIRED"
	ErrorCodeAgentTimeout    = "AGENT_TIMEOUT"
	ErrorCodeInvalidRequest  = "INVALID_REQUEST"
	ErrorCodeUnavailable     = "UNAVAILABLE"
	ErrorCodeUnknownProvider = "UNKNOWN_PROVIDER"
	ErrorCodeMissingConfig   = "MISSING_CONFIG"
	ErrorCodeRateLimited     = "RATE_LIMITED"
	ErrorCodeUnknownMethod   = "UNKNOWN_METHOD"
)

// ErrorShape represents a protocol error response
type ErrorShape struct {
	Code         string  `json:"code"`
	Message      string  `json:"message"`
	Details      []byte  `json:"details,omitempty"`
	Retryable    *bool   `json:"retryable,omitempty"`
	RetryAfterMs *uint64 `json:"retryAfterMs,omitempty"`
}

// Error makes ErrorShape implement the error interface
func (e ErrorShape) Error() string {
	return e.Message
}

// NewError creates a new ErrorShape
func NewError(code, message string) ErrorShape {
	return ErrorShape{
		Code:    code,
		Message: message,
	}
}

// FrameType represents the type of a frame
type FrameType string

const (
	// FrameTypeRequest is the request frame type
	FrameTypeRequest FrameType = "req"
	// FrameTypeResponse is the response frame type
	FrameTypeResponse FrameType = "res"
	// FrameTypeEvent is the event frame type
	FrameTypeEvent FrameType = "event"
)

// Frame is the discriminated union of all frame types
type Frame struct {
	// Type is the frame type discriminator
	Type FrameType `json:"type"`

	// Request is set when Type == "req"
	Request *RequestFrame `json:"req,omitempty"`

	// Response is set when Type == "res"
	Response *ResponseFrame `json:"res,omitempty"`

	// Event is set when Type == "event"
	Event *EventFrame `json:"event,omitempty"`
}

// RequestFrame represents a client → gateway RPC request
type RequestFrame struct {
	// Type is always "req"
	Type FrameType `json:"type"`
	// ID is the unique request identifier
	ID string `json:"id"`
	// Method is the RPC method name
	Method string `json:"method"`
	// Params are the method parameters (optional)
	Params []byte `json:"params,omitempty"`
}

// ResponseFrame represents a gateway → client RPC response
type ResponseFrame struct {
	// Type is always "res"
	Type FrameType `json:"type"`
	// ID matches the request ID
	ID string `json:"id"`
	// OK indicates success
	OK bool `json:"ok"`
	// Payload is the response data on success
	Payload []byte `json:"payload,omitempty"`
	// Error is set on failure
	Error *ErrorShape `json:"error,omitempty"`
}

// NewResponseOK creates a successful response frame
func NewResponseOK(id string, payload []byte) ResponseFrame {
	return ResponseFrame{
		Type:    FrameTypeResponse,
		ID:      id,
		OK:      true,
		Payload: payload,
	}
}

// NewResponseErr creates an error response frame
func NewResponseErr(id string, err ErrorShape) ResponseFrame {
	return ResponseFrame{
		Type:  FrameTypeResponse,
		ID:    id,
		OK:    false,
		Error: &err,
	}
}

// StateVersion represents state versioning for events
type StateVersion struct {
	Presence *uint64 `json:"presence,omitempty"`
	Health   *uint64 `json:"health,omitempty"`
}

// EventFrame represents a gateway → client server-push event
type EventFrame struct {
	// Type is always "event"
	Type FrameType `json:"type"`
	// Event is the event name
	Event string `json:"event"`
	// Payload is the event data
	Payload []byte `json:"payload,omitempty"`
	// Seq is the sequence number
	Seq *uint64 `json:"seq,omitempty"`
	// StateVersion contains state versioning info
	StateVersion *StateVersion `json:"stateVersion,omitempty"`
}

// NewEvent creates a new event frame
func NewEvent(event string, payload []byte, seq uint64) EventFrame {
	return EventFrame{
		Type:    FrameTypeEvent,
		Event:   event,
		Payload: payload,
		Seq:     &seq,
	}
}

// Event names
const (
	// Gateway events
	EventGatewayStart = "gateway.start"
	EventGatewayStop  = "gateway.stop"

	// Chat events
	EventChatThinking  = "chat.thinking"
	EventChatTextDelta = "chat.textDelta"
	EventChatToolStart = "chat.toolStart"
	EventChatToolDelta = "chat.toolDelta"
	EventChatToolEnd   = "chat.toolEnd"
	EventChatDone      = "chat.done"

	// Session events
	EventSessionStart = "session.start"
	EventSessionEnd   = "session.end"

	// Provider events
	EventProviderStatus = "provider.status"
	EventProviderError  = "provider.error"

	// Channel events
	EventChannelMessage = "channel.message"
	EventChannelReply   = "channel.reply"
)

// RPC method names
const (
	// Chat methods
	MethodChatSend    = "chat.send"
	MethodChatHistory = "chat.history"

	// Session methods
	MethodSessionList   = "session.list"
	MethodSessionCreate = "session.create"
	MethodSessionDelete = "session.delete"

	// Provider methods
	MethodProviderList      = "provider.list"
	MethodProviderConfigure = "provider.configure"
	MethodProviderTest      = "provider.test"

	// Auth methods
	MethodAuthStatus = "auth.status"
	MethodAuthSetup  = "auth.setup"
	MethodAuthLogin  = "auth.login"
	MethodAuthLogout = "auth.logout"

	// Config methods
	MethodConfigGet      = "config.get"
	MethodConfigSet      = "config.set"
	MethodConfigValidate = "config.validate"
)
