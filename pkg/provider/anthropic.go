// Package provider provides Anthropic (Claude) LLM implementation for Mortis
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

// NewAnthropic creates a new Anthropic provider
func NewAnthropic(cfg config.ProviderConfig) LLM {
	return &anthropicProvider{
		id:      "anthropic",
		name:    "Anthropic",
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		models:  cfg.Models,
		enabled: cfg.Enabled && cfg.APIKey != "",
		client:  http.DefaultClient,
	}
}

// anthropicProvider is the Anthropic LLM provider
type anthropicProvider struct {
	id      string
	name    string
	apiKey  string
	baseURL string
	model   string
	models  []string
	enabled bool
	client  *http.Client
}

// Info returns provider information
func (p *anthropicProvider) Info(ctx context.Context) (*Info, error) {
	features := []string{
		FeatureCompletion,
		FeatureStreaming,
		FeatureToolCalling,
		FeatureVision,
	}

	modelInfos := make([]ModelInfo, len(p.models))
	for i, m := range p.models {
		modelInfos[i] = ModelInfo{
			ID:          m,
			Name:        m,
			ContextSize: 200000,
		}
	}

	return &Info{
		ID:        p.id,
		Name:      p.name,
		Type:      "anthropic",
		Available: p.enabled,
		Models:    modelInfos,
		Features:  features,
	}, nil
}

// Complete generates a completion (non-streaming)
func (p *anthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.enabled {
		return nil, fmt.Errorf("anthropic provider: not enabled or missing API key")
	}

	// Use request model or default
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build Anthropic API request
	messages := p.convertMessages(req.Messages)

	// Combine system prompt with messages
	system := req.System

	anthropicReq := map[string]interface{}{
		"model":      model,
		"messages":   messages,
		"max_tokens": p.getMaxTokens(req),
	}

	if system != "" {
		anthropicReq["system"] = system
	}

	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		anthropicReq["top_p"] = req.TopP
	}

	// Marshal request
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	url := p.baseURL
	if url == "" {
		url = "https://api.anthropic.com/v1/messages"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var anthropicResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract text content
	var content string
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &CompletionResponse{
		Content:      content,
		FinishReason: anthropicResp.StopReason,
		TokensUsed:   anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
	}, nil
}

// Stream generates a streaming completion
func (p *anthropicProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamEvent, error) {
	if !p.enabled {
		return nil, fmt.Errorf("anthropic provider: not enabled or missing API key")
	}

	// Use request model or default
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build Anthropic API request
	messages := p.convertMessages(req.Messages)

	system := req.System

	anthropicReq := map[string]interface{}{
		"model":      model,
		"messages":   messages,
		"max_tokens": p.getMaxTokens(req),
		"stream":     true,
	}

	if system != "" {
		anthropicReq["system"] = system
	}

	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		anthropicReq["top_p"] = req.TopP
	}

	// Marshal request
	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	url := p.baseURL
	if url == "" {
		url = "https://api.anthropic.com/v1/messages"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

			var chunk struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
				Message struct {
					StopReason string `json:"stop_reason"`
				} `json:"message"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			switch chunk.Type {
			case "content_block_delta":
				if chunk.Delta.Text != "" {
					eventCh <- StreamEvent{
						Type:    StreamEventTypeTextDelta,
						Content: chunk.Delta.Text,
					}
				}
			case "message_stop":
				eventCh <- StreamEvent{
					Type:         StreamEventTypeDone,
					FinishReason: chunk.Message.StopReason,
				}
				return
			}
		}
	}()

	return eventCh, nil
}

// Close closes any resources
func (p *anthropicProvider) Close() error {
	return nil
}

// convertMessages converts provider messages to Anthropic format
func (p *anthropicProvider) convertMessages(messages []Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		// Anthropic doesn't support system role in messages array
		if msg.Role == "system" {
			continue
		}
		result = append(result, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return result
}

// getMaxTokens returns the max tokens to generate
func (p *anthropicProvider) getMaxTokens(req CompletionRequest) int {
	if req.MaxTokens > 0 {
		return req.MaxTokens
	}
	return DefaultMaxTokens
}
