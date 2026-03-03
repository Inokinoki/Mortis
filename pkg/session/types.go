// Package session provides session management for Mortis
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Inokinoki/mortis/pkg/protocol"
	"github.com/google/uuid"
)

// Session represents a conversation session
type Session struct {
	// ID is the unique session identifier
	ID string `json:"id"`
	// Name is the session name
	Name string `json:"name"`
	// CreatedAt is the creation timestamp
	CreatedAt int64 `json:"createdAt"`
	// UpdatedAt is the last update timestamp
	UpdatedAt int64 `json:"updatedAt"`
	// MessageCount is the number of messages
	MessageCount int `json:"messageCount"`
	// Model is the default model for this session
	Model string `json:"model"`
	// Provider is the default provider for this session
	Provider string `json:"provider"`
	// ProjectID is the optional project/workspace ID
	ProjectID string `json:"projectId,omitempty"`
	// ChannelID is the optional channel binding
	ChannelID string `json:"channelId,omitempty"`
	// Mutex protects concurrent access
	mu sync.RWMutex
}

// Manager manages sessions
type Manager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	dataDir  string
}

// NewManager creates a new session manager
func NewManager(dataDir string) *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		dataDir:  dataDir,
	}
}

// Create creates a new session
func (m *Manager) Create(name, model, provider string) (*Session, error) {
	id := uuid.New().String()
	sess := &Session{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Model:     model,
		Provider:  provider,
	}

	// Create session directory
	sessionDir := filepath.Join(m.dataDir, sess.ID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, err
	}

	// Create messages file
	messagesFile := filepath.Join(sessionDir, "messages.jsonl")
	if err := os.WriteFile(messagesFile, []byte{}, 0o644); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.sessions[sess.ID] = sess
	m.mu.Unlock()

	return sess, nil
}

// Get returns a session by ID
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[id]
	return sess, ok
}

// List returns all sessions
func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Session, 0, len(m.sessions))
	for _, sess := range m.sessions {
		result = append(result, sess)
	}
	return result
}

// Delete deletes a session
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[id]; !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	delete(m.sessions, id)

	// Remove session directory
	sessionDir := filepath.Join(m.dataDir, id)
	return os.RemoveAll(sessionDir)
}

// AddMessage adds a message to a session
func (m *Manager) AddMessage(ctx context.Context, sessionID string, msg protocol.Message) error {
	sess, ok := m.Get(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Serialize message
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	// Append to messages file
	sessionDir := filepath.Join(m.dataDir, sess.ID)
	messagesFile := filepath.Join(sessionDir, "messages.jsonl")

	f, err := os.OpenFile(messagesFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}

	sess.mu.Lock()
	sess.MessageCount++
	sess.UpdatedAt = time.Now().Unix()
	sess.mu.Unlock()

	return nil
}

// GetMessages retrieves messages from a session
func (m *Manager) GetMessages(ctx context.Context, sessionID string, limit int, before string) ([]protocol.Message, error) {
	sess, ok := m.Get(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	sessionDir := filepath.Join(m.dataDir, sess.ID)
	messagesFile := filepath.Join(sessionDir, "messages.jsonl")

	data, err := os.ReadFile(messagesFile)
	if err != nil {
		return nil, err
	}

	// Parse JSONL
	var messages []protocol.Message
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var msg protocol.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// Apply limit
	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

// Compact compacts a session by summarizing old messages
func (m *Manager) Compact(ctx context.Context, sessionID string, threshold float64) error {
	sess, ok := m.Get(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	sess.mu.RLock()
	messageCount := sess.MessageCount
	sess.mu.RUnlock()

	// Calculate if compaction is needed
	if threshold <= 0 || float64(messageCount) < threshold*float64(200) {
		return nil // No compaction needed
	}

	// Truncate to last N messages (simple compaction without LLM)
	// Full implementation would use LLM to summarize old messages
	messages, err := m.GetMessages(ctx, sessionID, 100, "")
	if err != nil {
		return err
	}

	// Write back compacted messages
	sessionDir := filepath.Join(m.dataDir, sess.ID)
	messagesFile := filepath.Join(sessionDir, "messages.jsonl")

	f, err := os.Create(messagesFile)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if _, err := f.Write(data); err != nil {
			return err
		}
	}

	sess.mu.Lock()
	sess.MessageCount = len(messages)
	sess.UpdatedAt = time.Now().Unix()
	sess.mu.Unlock()

	return nil
}

// splitLines splits a byte slice into lines
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
