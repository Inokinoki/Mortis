// Package provider provides Local LLM implementation for Mortis
// Supports Ollama, LM Studio, and other OpenAI-compatible local servers
package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Inokinoki/mortis/pkg/config"
)

// NewLocalLLM creates a new Local LLM provider
func NewLocalLLM(cfg config.ProviderConfig) LLM {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1/chat/completions" // Default to Ollama
	}
	return &localProvider{
		id:      "local",
		name:    "Local",
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		models:  cfg.Models,
		enabled: cfg.Enabled,
		client:  http.DefaultClient,
	}
}

// localProvider is the Local LLM provider
type localProvider struct {
	id      string
	name    string
	baseURL string
	apiKey  string
	model   string
	models  []string
	enabled bool
	client  *http.Client
}

// Info returns provider information
func (p *localProvider) Info(ctx context.Context) (*Info, error) {
	features := []string{
		FeatureCompletion,
		FeatureStreaming,
	}

	modelInfos := make([]ModelInfo, len(p.models))
	for i, m := range p.models {
		modelInfos[i] = ModelInfo{
			ID:          m,
			Name:        m,
			ContextSize: DefaultContextSize,
		}
	}

	return &Info{
		ID:        p.id,
		Name:      p.name,
		Type:      "local",
		Available: p.enabled,
		Models:    modelInfos,
		Features:  features,
	}, nil
}

// Complete generates a completion (non-streaming)
func (p *localProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("local provider: not enabled")
	}

	// Use request model or default
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build API request (OpenAI-compatible format)
	apiReq := map[string]interface{}{
		"model":    model,
		"messages": p.convertMessages(req.Messages),
	}

	if req.System != "" {
		apiReq["messages"] = append([]map[string]interface{}{
			{"role": "system", "content": req.System},
		}, apiReq["messages"].([]map[string]interface{})...)
	}

	if req.MaxTokens > 0 {
		apiReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		apiReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		apiReq["top_p"] = req.TopP
	}

	// Marshal request
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response (OpenAI-compatible format)
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &CompletionResponse{
		Content:      apiResp.Choices[0].Message.Content,
		FinishReason: apiResp.Choices[0].FinishReason,
		TokensUsed:   apiResp.Usage.TotalTokens,
	}, nil
}

// Stream generates a streaming completion
func (p *localProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, error) {
	if !p.enabled {
		return nil, fmt.Errorf("local provider: not enabled")
	}

	// Use request model or default
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build API request (OpenAI-compatible format)
	apiReq := map[string]interface{}{
		"model":    model,
		"messages": p.convertMessages(req.Messages),
		"stream":   true,
	}

	if req.System != "" {
		apiReq["messages"] = append([]map[string]interface{}{
			{"role": "system", "content": req.System},
		}, apiReq["messages"].([]map[string]interface{})...)
	}

	if req.MaxTokens > 0 {
		apiReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		apiReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		apiReq["top_p"] = req.TopP
	}

	// Marshal request
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Create event channel
	eventCh := make(chan StreamEvent, 16)

	go func() {
		defer resp.Body.Close()
		defer close(eventCh)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				eventCh <- StreamEvent{Type: StreamEventTypeDone}
				return
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
					FinishReason *string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 {
				if chunk.Choices[0].Delta.Content != "" {
					eventCh <- StreamEvent{
						Type:    StreamEventTypeTextDelta,
						Content: chunk.Choices[0].Delta.Content,
					}
				}
				if chunk.Choices[0].FinishReason != nil {
					eventCh <- StreamEvent{
						Type:         StreamEventTypeDone,
						FinishReason: *chunk.Choices[0].FinishReason,
					}
					return
				}
			}
		}
	}()

	return eventCh, nil
}

// Close closes any resources
func (p *localProvider) Close() error {
	return nil
}

// convertMessages converts provider messages to API format
func (p *localProvider) convertMessages(messages []Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		result = append(result, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return result
}
