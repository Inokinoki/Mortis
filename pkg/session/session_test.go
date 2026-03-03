// Package session provides session management tests
package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Inokinoki/mortis/pkg/protocol"
	"github.com/google/uuid"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	if mgr == nil {
		t.Fatal("expected manager to be created")
	}

	if mgr.dataDir != tmpDir {
		t.Errorf("expected data dir %s, got %s", tmpDir, mgr.dataDir)
	}
}

func TestManagerCreate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if sess == nil {
		t.Fatal("expected session to be created")
	}

	if sess.ID == "" {
		t.Error("expected session ID to be set")
	}

	if sess.Name != "test-session" {
		t.Errorf("expected name 'test-session', got %s", sess.Name)
	}

	if sess.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %s", sess.Model)
	}

	if sess.Provider != "openai" {
		t.Errorf("expected provider 'openai', got %s", sess.Provider)
	}

	if sess.MessageCount != 0 {
		t.Errorf("expected 0 messages, got %d", sess.MessageCount)
	}

	if sess.CreatedAt == 0 {
		t.Error("expected creation time to be set")
	}

	if sess.UpdatedAt == 0 {
		t.Error("expected update time to be set")
	}
}

func TestManagerGet(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	created, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	retrieved, ok := mgr.Get(created.ID)
	if !ok {
		t.Fatalf("session not found: %s", created.ID)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, created.ID)
	}

	if retrieved.Name != created.Name {
		t.Errorf("name mismatch: got %s, want %s", retrieved.Name, created.Name)
	}
}

func TestManagerGetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, ok := mgr.Get("nonexistent-id")
	if ok {
		t.Error("expected false for nonexistent session")
	}
}

func TestManagerList(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, _ = mgr.Create("session-1", "gpt-4o", "openai")
	_, _ = mgr.Create("session-2", "claude-3", "anthropic")
	_, _ = mgr.Create("session-3", "gpt-4o-mini", "openai")

	sessions := mgr.List()

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}

	sessionIDs := make(map[string]bool)
	for _, sess := range sessions {
		sessionIDs[sess.ID] = true
	}

	if len(sessionIDs) != 3 {
		t.Errorf("expected 3 unique session IDs, got %d", len(sessionIDs))
	}
}

func TestManagerListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sessions := mgr.List()

	if sessions == nil {
		t.Fatal("expected non-nil sessions list")
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestManagerDelete(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	sessionDir := filepath.Join(tmpDir, sess.ID)
	if _, err := os.Stat(sessionDir); err != nil {
		t.Fatalf("session directory not created: %v", err)
	}

	err = mgr.Delete(sess.ID)
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	if _, err := os.Stat(sessionDir); !os.IsNotExist(err) {
		t.Error("session directory should be deleted")
	}

	_, ok := mgr.Get(sess.ID)
	if ok {
		t.Error("session should be removed from manager")
	}
}

func TestManagerDeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.Delete("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestAddMessage(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	msg := protocol.Message{
		ID:        uuid.New().String(),
		Role:      "user",
		Content:   "Hello, world!",
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()
	if err := mgr.AddMessage(ctx, sess.ID, msg); err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	retrieved, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("session should still exist")
	}

	if retrieved.MessageCount != 1 {
		t.Errorf("expected 1 message, got %d", retrieved.MessageCount)
	}
}

func TestAddMessageToNonExistentSession(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	msg := protocol.Message{
		ID:        uuid.New().String(),
		Role:      "user",
		Content:   "Hello",
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()
	err := mgr.AddMessage(ctx, "nonexistent-id", msg)

	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestGetMessages(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	messages := []protocol.Message{
		{
			ID:        uuid.New().String(),
			Role:      "user",
			Content:   "Message 1",
			Timestamp: time.Now().Unix(),
		},
		{
			ID:        uuid.New().String(),
			Role:      "assistant",
			Content:   "Response 1",
			Timestamp: time.Now().Unix(),
		},
		{
			ID:        uuid.New().String(),
			Role:      "user",
			Content:   "Message 2",
			Timestamp: time.Now().Unix(),
		},
	}

	ctx := context.Background()
	for _, msg := range messages {
		if err := mgr.AddMessage(ctx, sess.ID, msg); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	retrieved, err := mgr.GetMessages(ctx, sess.ID, 0, "")
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}

	if len(retrieved) != len(messages) {
		t.Errorf("expected %d messages, got %d", len(messages), len(retrieved))
	}
}

func TestGetMessagesWithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	messages := []protocol.Message{}
	for i := 0; i < 10; i++ {
		messages = append(messages, protocol.Message{
			ID:        uuid.New().String(),
			Role:      "user",
			Content:   string(rune('a' + i)),
			Timestamp: time.Now().Unix(),
		})
	}

	ctx := context.Background()
	for _, msg := range messages {
		if err := mgr.AddMessage(ctx, sess.ID, msg); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	retrieved, err := mgr.GetMessages(ctx, sess.ID, 5, "")
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}

	if len(retrieved) != 5 {
		t.Errorf("expected 5 messages with limit, got %d", len(retrieved))
	}
}

func TestGetMessagesEmptySession(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	ctx := context.Background()
	messages, err := mgr.GetMessages(ctx, sess.ID, 0, "")
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}

	if messages == nil {
		t.Fatalf("expected non-nil messages list, got nil")
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(messages))
	}
}

func TestCompact(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	ctx := context.Background()
	for i := 0; i < 150; i++ {
		msg := protocol.Message{
			ID:        uuid.New().String(),
			Role:      "user",
			Content:   "Message",
			Timestamp: time.Now().Unix(),
		}
		if err := mgr.AddMessage(ctx, sess.ID, msg); err != nil {
			t.Fatalf("failed to add message: %v", err)
		}
	}

	updated, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("session should exist")
	}

	if updated.MessageCount != 150 {
		t.Errorf("expected 150 messages before compaction, got %d", updated.MessageCount)
	}

	if err := mgr.Compact(ctx, sess.ID, 0.5); err != nil {
		t.Fatalf("failed to compact: %v", err)
	}

	compacted, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("session should still exist")
	}

	if compacted.MessageCount > 100 {
		t.Errorf("expected max 100 messages after compaction, got %d", compacted.MessageCount)
	}
}

func TestCompactBelowThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	ctx := context.Background()
	msg := protocol.Message{
		ID:        uuid.New().String(),
		Role:      "user",
		Content:   "Message",
		Timestamp: time.Now().Unix(),
	}

	if err := mgr.AddMessage(ctx, sess.ID, msg); err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	if err := mgr.Compact(ctx, sess.ID, 1.0); err != nil {
		t.Fatalf("failed to compact: %v", err)
	}

	updated, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("session should exist")
	}

	if updated.MessageCount != 1 {
		t.Errorf("expected 1 message (no compaction), got %d", updated.MessageCount)
	}
}

func TestCompactNonExistentSession(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	ctx := context.Background()
	err := mgr.Compact(ctx, "nonexistent-id", 0.5)

	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestSessionConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 5; i++ {
		go func(id int) {
			msg := protocol.Message{
				ID:        uuid.New().String(),
				Role:      "user",
				Content:   string(rune('a' + id)),
				Timestamp: time.Now().Unix(),
			}

			ctx := context.Background()
			err := mgr.AddMessage(ctx, sess.ID, msg)
			errors <- err
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
		if err := <-errors; err != nil {
			t.Errorf("concurrent add failed: %v", err)
		}
	}

	retrieved, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("session should still exist")
	}

	if retrieved.MessageCount != 5 {
		t.Errorf("expected 5 messages after concurrent adds, got %d", retrieved.MessageCount)
	}
}

func TestSessionWithDifferentRoles(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	ctx := context.Background()
	roles := []string{"system", "user", "assistant", "tool"}
	for _, role := range roles {
		msg := protocol.Message{
			ID:        uuid.New().String(),
			Role:      role,
			Content:   "test content",
			Timestamp: time.Now().Unix(),
		}
		if err := mgr.AddMessage(ctx, sess.ID, msg); err != nil {
			t.Fatalf("failed to add %s message: %v", role, err)
		}
	}

	messages, err := mgr.GetMessages(ctx, sess.ID, 0, "")
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}

	if len(messages) != len(roles) {
		t.Errorf("expected %d messages, got %d", len(roles), len(messages))
	}

	for i, msg := range messages {
		if msg.Role != roles[i] {
			t.Errorf("message %d: expected role %s, got %s", i, roles[i], msg.Role)
		}
	}
}

func TestSessionProjectIDAndChannelID(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	sess.ProjectID = "project-123"
	sess.ChannelID = "channel-456"

	retrieved, ok := mgr.Get(sess.ID)
	if !ok {
		t.Fatal("session should exist")
	}

	if retrieved.ProjectID != sess.ProjectID {
		t.Errorf("project ID mismatch")
	}

	if retrieved.ChannelID != sess.ChannelID {
		t.Errorf("channel ID mismatch")
	}
}

func TestSessionFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	sess, err := mgr.Create("test-session", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	sessionDir := filepath.Join(tmpDir, sess.ID)
	info, err := os.Stat(sessionDir)
	if err != nil {
		t.Fatalf("session directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("session directory should be a directory")
	}

	messagesFile := filepath.Join(sessionDir, "messages.jsonl")
	info, err = os.Stat(messagesFile)
	if err != nil {
		t.Fatalf("messages file not created: %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Error("messages file should be regular file")
	}
}

func TestMultipleManagers(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	mgr1 := NewManager(tmpDir1)
	mgr2 := NewManager(tmpDir2)

	sess1, _ := mgr1.Create("session-1", "gpt-4o", "openai")
	sess2, _ := mgr2.Create("session-2", "claude-3", "anthropic")

	_, ok1 := mgr1.Get(sess1.ID)
	_, ok2 := mgr2.Get(sess1.ID)

	if !ok1 {
		t.Error("mgr1 should have session-1")
	}

	if ok2 {
		t.Error("mgr2 should not have session-1 (different manager)")
	}

	_, ok1 = mgr2.Get(sess2.ID)
	_, ok2 = mgr1.Get(sess2.ID)

	if !ok1 {
		t.Error("mgr2 should have session-2")
	}

	if ok2 {
		t.Error("mgr1 should not have session-2 (different manager)")
	}
}
