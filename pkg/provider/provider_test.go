// Package provider tests LLM provider abstractions
package provider

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockLLM struct {
	info   *Info
	closed bool
}

func (m *mockLLM) Info(ctx context.Context) (*Info, error) {
	return m.info, nil
}

func (m *mockLLM) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{
		Content:      "Mock response",
		FinishReason: "stop",
		TokensUsed:   100,
	}, nil
}

func (m *mockLLM) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, 1)
	go func() {
		ch <- StreamEvent{
			Type:    StreamEventTypeTextDelta,
			Content: "Mock",
		}
		ch <- StreamEvent{
			Type:         StreamEventTypeDone,
			FinishReason: "stop",
			TokensUsed:   100,
		}
		close(ch)
	}()
	return ch, nil
}

func (m *mockLLM) Close() error {
	m.closed = true
	return nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("expected registry to be created")
	}

	if registry.providers == nil {
		t.Error("expected providers map to be initialized")
	}
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	mock := &mockLLM{
		info: &Info{
			ID:     "mock",
			Name:   "Mock LLM",
			Type:   "mock",
			Models: []ModelInfo{{ID: "model-1"}},
		},
	}

	registry.Register("mock", mock)

	retrieved, ok := registry.Get("mock")
	if !ok {
		t.Error("expected mock provider to be registered")
	}

	if retrieved != mock {
		t.Error("retrieved provider should be the same instance")
	}
}

func TestRegistryUnregister(t *testing.T) {
	registry := NewRegistry()

	mock := &mockLLM{
		info: &Info{
			ID:   "mock",
			Name: "Mock LLM",
			Type: "mock",
		},
	}

	registry.Register("mock", mock)

	registry.Unregister("mock")

	_, ok := registry.Get("mock")
	if ok {
		t.Error("expected mock provider to be unregistered")
	}
}

func TestRegistryGetNonExistent(t *testing.T) {
	registry := NewRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent provider")
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	mock1 := &mockLLM{
		info: &Info{
			ID:   "mock1",
			Name: "Mock 1",
			Type: "mock",
		},
	}
	mock2 := &mockLLM{
		info: &Info{
			ID:   "mock2",
			Name: "Mock 2",
			Type: "mock",
		},
	}

	registry.Register("mock1", mock1)
	registry.Register("mock2", mock2)

	list := registry.List()

	if len(list) != 2 {
		t.Errorf("expected 2 providers, got %d", len(list))
	}

	if _, ok := list["mock1"]; !ok {
		t.Error("expected mock1 in list")
	}

	if _, ok := list["mock2"]; !ok {
		t.Error("expected mock2 in list")
	}
}

func TestRegistrySetDefault(t *testing.T) {
	registry := NewRegistry()

	mock := &mockLLM{
		info: &Info{
			ID:   "mock",
			Name: "Mock",
			Type: "mock",
		},
	}

	registry.Register("mock", mock)
	registry.SetDefault("mock")

	retrieved, ok := registry.GetDefault()
	if !ok {
		t.Error("expected default provider to be set")
	}

	if retrieved != mock {
		t.Error("retrieved default should be the mock provider")
	}
}

func TestRegistryGetDefaultNotSet(t *testing.T) {
	registry := NewRegistry()

	_, ok := registry.GetDefault()
	if ok {
		t.Error("expected false when default not set")
	}
}

func TestRegistryConcurrency(t *testing.T) {
	registry := NewRegistry()

	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			mock := &mockLLM{
				info: &Info{
					ID:   uuid.New().String(),
					Name: "Mock",
					Type: "mock",
				},
			}
			registry.Register(uuid.New().String(), mock)
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	list := registry.List()
	if len(list) != 5 {
		t.Errorf("expected 5 providers after concurrent registration, got %d", len(list))
	}
}

func TestMessageTypes(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{"system", true},
		{"user", true},
		{"assistant", true},
		{"tool", true},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			msg := Message{Role: tt.role, Content: "test"}

			if tt.valid {
				if msg.Role != tt.role {
					t.Error("role should be preserved")
				}
			}
		})
	}
}

func TestToolCall(t *testing.T) {
	tool := ToolCall{
		ID:        uuid.New().String(),
		Name:      "web_search",
		Arguments: []byte(`{"query":"test"}`),
	}

	if tool.ID == "" {
		t.Error("expected tool call ID to be set")
	}

	if tool.Name != "web_search" {
		t.Error("tool name mismatch")
	}

	if len(tool.Arguments) == 0 {
		t.Error("expected arguments to be set")
	}

	if tool.Result != "" {
		t.Error("expected empty result (not yet executed)")
	}
}

func TestToolResult(t *testing.T) {
	result := ToolResult{
		ID:      uuid.New().String(),
		Success: true,
		Result:  "Tool output",
	}

	if result.ID == "" {
		t.Error("expected tool result ID to be set")
	}

	if !result.Success {
		t.Error("expected success to be true")
	}

	if result.Result != "Tool output" {
		t.Error("result content mismatch")
	}
}

func TestCompletionRequest(t *testing.T) {
	req := CompletionRequest{
		SessionID:    "session-123",
		MessageID:    uuid.New().String(),
		Messages:     []Message{{Role: "user", Content: "Hello"}},
		System:       "You are a helpful assistant.",
		Model:        "gpt-4o",
		ToolsEnabled: true,
		MaxTokens:    1000,
		Temperature:  0.7,
		TopP:         1.0,
	}

	if req.SessionID != "session-123" {
		t.Error("session ID mismatch")
	}

	if req.MaxTokens != 1000 {
		t.Error("max tokens mismatch")
	}

	if req.Temperature != 0.7 {
		t.Error("temperature mismatch")
	}

	if req.TopP != 1.0 {
		t.Error("top_p mismatch")
	}
}

func TestCompletionResponse(t *testing.T) {
	resp := CompletionResponse{
		Content:      "Generated text",
		FinishReason: "stop",
		TokensUsed:   150,
		ToolCalls: []ToolCall{
			{
				ID:        uuid.New().String(),
				Name:      "tool",
				Arguments: []byte(`{}`),
			},
		},
	}

	if resp.Content != "Generated text" {
		t.Error("content mismatch")
	}

	if resp.FinishReason != "stop" {
		t.Error("finish reason mismatch")
	}

	if resp.TokensUsed != 150 {
		t.Error("tokens used mismatch")
	}

	if len(resp.ToolCalls) != 1 {
		t.Error("expected 1 tool call")
	}
}

func TestStreamEventTypes(t *testing.T) {
	tests := []struct {
		name   string
		event  StreamEvent
		isText bool
		isTool bool
		isDone bool
	}{
		{
			name:   "text delta",
			event:  StreamEvent{Type: StreamEventTypeTextDelta, Content: "Partial"},
			isText: true,
			isTool: false,
			isDone: false,
		},
		{
			name:   "tool call",
			event:  StreamEvent{Type: StreamEventTypeToolCall, ToolCall: &ToolCall{ID: "1", Name: "tool"}},
			isText: false,
			isTool: true,
			isDone: false,
		},
		{
			name:   "tool delta",
			event:  StreamEvent{Type: StreamEventTypeToolDelta, ToolDelta: &ToolDelta{ID: "1", Delta: "out", Done: true}},
			isText: false,
			isTool: false,
			isDone: false,
		},
		{
			name:   "done",
			event:  StreamEvent{Type: StreamEventTypeDone, FinishReason: "stop", TokensUsed: 100},
			isText: false,
			isTool: false,
			isDone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event.Type == StreamEventTypeTextDelta && tt.event.Content == "" {
				t.Error("text delta should have content")
			}

			if tt.event.Type == StreamEventTypeToolCall && tt.event.ToolCall == nil {
				t.Error("tool call should be set")
			}

			if tt.event.Type == StreamEventTypeDone && tt.event.FinishReason == "" {
				t.Error("done event should have finish reason")
			}
		})
	}
}

func TestModelInfo(t *testing.T) {
	info := ModelInfo{
		ID:          "gpt-4o",
		Name:        "GPT-4o",
		ContextSize: 128000,
		Description: "A powerful multimodal model",
	}

	if info.ID != "gpt-4o" {
		t.Error("ID mismatch")
	}

	if info.Name != "GPT-4o" {
		t.Error("name mismatch")
	}

	if info.ContextSize != 128000 {
		t.Error("context size mismatch")
	}

	if info.Description != "A powerful multimodal model" {
		t.Error("description mismatch")
	}
}

func TestProviderInfo(t *testing.T) {
	tests := []struct {
		name string
		info *Info
	}{
		{
			name: "minimal info",
			info: &Info{
				ID:   "provider-1",
				Name: "Provider 1",
				Type: "custom",
			},
		},
		{
			name: "full info",
			info: &Info{
				ID:        "provider-2",
				Name:      "Provider 2",
				Type:      "openai",
				Available: true,
				Models: []ModelInfo{
					{ID: "gpt-4o", ContextSize: 128000},
					{ID: "gpt-4o-mini", ContextSize: 128000},
				},
				Features: []string{
					FeatureCompletion,
					FeatureStreaming,
					FeatureToolCalling,
					FeatureVision,
					FeatureEmbeddings,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.info.ID != "provider-1" && tt.info.ID != "provider-2" {
				t.Error("ID should be set")
			}

			if len(tt.info.Models) > 0 {
				for _, model := range tt.info.Models {
					if model.ContextSize == 0 {
						t.Error("models should have context sizes set")
					}
				}
			}
		})
	}
}

func TestToolDelta(t *testing.T) {
	delta := ToolDelta{
		ID:    "call-123",
		Delta: "partial output",
		Done:  false,
	}

	if delta.ID != "call-123" {
		t.Error("ID mismatch")
	}

	if delta.Done {
		t.Error("expected done to be false")
	}

	if delta.Delta != "partial output" {
		t.Error("delta mismatch")
	}

	delta.Done = true
	if !delta.Done {
		t.Error("done should be true after update")
	}
}

func TestLLMInterface(t *testing.T) {
	mock := &mockLLM{
		info: &Info{
			ID:   "mock",
			Name: "Mock LLM",
			Type: "mock",
		},
	}

	ctx := context.Background()
	info, err := mock.Info(ctx)
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	if info.ID != "mock" {
		t.Error("info ID mismatch")
	}

	if mock.closed {
		t.Error("provider should not be closed initially")
	}

	if err := mock.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	if !mock.closed {
		t.Error("provider should be closed after Close()")
	}
}

func TestCompletionStreaming(t *testing.T) {
	mock := &mockLLM{
		info: &Info{
			ID:   "mock",
			Name: "Mock LLM",
			Type: "mock",
		},
	}

	req := CompletionRequest{
		SessionID: "session-123",
		MessageID: uuid.New().String(),
		Messages:  []Message{{Role: "user", Content: "Hello"}},
	}

	ctx := context.Background()
	ch, err := mock.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}

	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	received := 0
	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatal("stream timed out")
		}

		if received >= 2 {
			break
		}
	}

	if received != 2 {
		t.Errorf("expected 2 events, got %d", received)
	}
}

func TestConstants(t *testing.T) {
	constants := []struct {
		name  string
		value string
	}{
		{"FeatureCompletion", FeatureCompletion},
		{"FeatureStreaming", FeatureStreaming},
		{"FeatureToolCalling", FeatureToolCalling},
		{"FeatureVision", FeatureVision},
		{"FeatureEmbeddings", FeatureEmbeddings},
	}

	for _, tt := range constants {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("constant %s should not be empty", tt.name)
			}
		})
	}

	if DefaultContextSize != 128000 {
		t.Errorf("expected default context 128000, got %d", DefaultContextSize)
	}

	if DefaultMaxTokens != 4096 {
		t.Errorf("expected default max tokens 4096, got %d", DefaultMaxTokens)
	}

	if DefaultTemperature != 0.7 {
		t.Errorf("expected default temperature 0.7, got %f", DefaultTemperature)
	}

	if DefaultTopP != 1.0 {
		t.Errorf("expected default top_p 1.0, got %f", DefaultTopP)
	}
}

func TestMultipleProvidersSameType(t *testing.T) {
	registry := NewRegistry()

	mock1 := &mockLLM{
		info: &Info{
			ID:   "mock1",
			Name: "Mock 1",
			Type: "openai",
		},
	}
	mock2 := &mockLLM{
		info: &Info{
			ID:   "mock2",
			Name: "Mock 2",
			Type: "openai",
		},
	}

	registry.Register("mock1", mock1)
	registry.Register("mock2", mock2)

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("expected 2 providers, got %d", len(list))
	}
}

func TestProviderOverride(t *testing.T) {
	registry := NewRegistry()

	mock1 := &mockLLM{
		info: &Info{
			ID:   "mock",
			Name: "Mock",
			Type: "mock",
		},
	}
	mock2 := &mockLLM{
		info: &Info{
			ID:   "mock",
			Name: "Mock Updated",
			Type: "mock",
		},
	}

	registry.Register("mock", mock1)
	registry.Register("mock", mock2)

	retrieved, ok := registry.Get("mock")
	if !ok {
		t.Fatal("expected provider to be registered")
	}

	if retrieved != mock2 {
		t.Error("second registration should override first")
	}
}
