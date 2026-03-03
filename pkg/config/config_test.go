// Package config provides configuration management tests
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected default host 127.0.0.1, got %s", cfg.Server.Host)
	}

	if cfg.Server.Port != 13131 {
		t.Errorf("expected default port 13131, got %d", cfg.Server.Port)
	}

	if !cfg.Auth.Disabled {
		t.Error("expected auth to be disabled by default")
	}

	if cfg.Gateway.DefaultProvider != "openai" {
		t.Errorf("expected default provider openai, got %s", cfg.Gateway.DefaultProvider)
	}

	if cfg.Gateway.DefaultModel != "gpt-4o" {
		t.Errorf("expected default model gpt-4o, got %s", cfg.Gateway.DefaultModel)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	original := DefaultConfig()
	original.Server.Host = "0.0.0.0"
	original.Server.Port = 8080
	original.Gateway.Name = "TestGateway"

	if err := original.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Server.Host != original.Server.Host {
		t.Errorf("host mismatch: got %s, want %s", loaded.Server.Host, original.Server.Host)
	}

	if loaded.Server.Port != original.Server.Port {
		t.Errorf("port mismatch: got %d, want %d", loaded.Server.Port, original.Server.Port)
	}

	if loaded.Gateway.Name != original.Gateway.Name {
		t.Errorf("name mismatch: got %s, want %s", loaded.Gateway.Name, original.Gateway.Name)
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.json")

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}

	if !os.IsNotExist(err) {
		t.Errorf("expected file not found error, got %v", err)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	invalidContent := []byte(`{"server": {"host": "invalid"`)
	if err := os.WriteFile(configPath, invalidContent, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestConfigMarshalRoundTrip(t *testing.T) {
	original := DefaultConfig()
	original.Providers = map[string]ProviderConfig{
		"test": {
			Type:    "openai",
			Model:   "gpt-4o",
			Enabled: true,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(loaded.Providers) != len(original.Providers) {
		t.Errorf("provider count mismatch: got %d, want %d", len(loaded.Providers), len(original.Providers))
	}
}

func TestProviderConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   ProviderConfig
		expected string
	}{
		{
			name: "openai provider",
			config: ProviderConfig{
				Type:    "openai",
				Model:   "gpt-4o",
				Enabled: true,
			},
			expected: "openai",
		},
		{
			name: "anthropic provider",
			config: ProviderConfig{
				Type:    "anthropic",
				Model:   "claude-3-opus",
				Enabled: true,
			},
			expected: "anthropic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Type != tt.expected {
				t.Errorf("provider type: got %s, want %s", tt.config.Type, tt.expected)
			}
		})
	}
}

func TestSessionConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := SessionConfig{
		DataDir:            tmpDir,
		AutoCompactEnabled: true,
		CompactThreshold:   0.95,
		HistoryLimit:       200,
	}

	if !cfg.AutoCompactEnabled {
		t.Error("expected auto compact to be enabled")
	}

	if cfg.CompactThreshold != 0.95 {
		t.Errorf("expected threshold 0.95, got %f", cfg.CompactThreshold)
	}

	if cfg.HistoryLimit != 200 {
		t.Errorf("expected history limit 200, got %d", cfg.HistoryLimit)
	}
}

func TestToolsConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Tools.Exec.Enabled {
		t.Error("expected exec tool to be enabled by default")
	}

	if cfg.Tools.Exec.Sandbox.Backend != "disabled" {
		t.Errorf("expected sandbox backend disabled, got %s", cfg.Tools.Exec.Sandbox.Backend)
	}

	if !cfg.Tools.WebFetch.Enabled {
		t.Error("expected web fetch tool to be enabled by default")
	}
}

func TestMemoryConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Memory == nil {
		t.Error("expected memory config to be present")
	}

	mem := *cfg.Memory
	if !mem.Enabled {
		t.Error("expected memory to be enabled by default")
	}

	if mem.Provider != "sqlite" {
		t.Errorf("expected provider sqlite, got %s", mem.Provider)
	}

	if mem.ChunkSize != 512 {
		t.Errorf("expected chunk size 512, got %d", mem.ChunkSize)
	}
}

func TestAuthConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Auth.Disabled {
		t.Error("expected auth to be disabled by default")
	}

	tests := []struct {
		name string
		set  func(*AuthConfig)
	}{
		{
			name: "with password",
			set: func(ac *AuthConfig) {
				ac.PasswordHash = "hashed_password"
			},
		},
		{
			name: "with API keys",
			set: func(ac *AuthConfig) {
				ac.APIKeys = []string{"key1", "key2"}
			},
		},
		{
			name: "with passkeys",
			set: func(ac *AuthConfig) {
				ac.Passkeys = []PasskeyConfig{
					{ID: "key1", Name: "Test Key"},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.set(&cfg.Auth)
		})
	}
}

func TestChannelConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   ChannelConfig
		expected string
	}{
		{
			name: "telegram channel",
			config: ChannelConfig{
				Type:    "telegram",
				Enabled: true,
				Token:   "bot_token",
			},
			expected: "telegram",
		},
		{
			name: "discord channel",
			config: ChannelConfig{
				Type:    "discord",
				Enabled: true,
				Token:   "bot_token",
			},
			expected: "discord",
		},
		{
			name: "slack channel",
			config: ChannelConfig{
				Type:       "slack",
				Enabled:    true,
				WebhookURL: "https://hooks.slack.com",
			},
			expected: "slack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Type != tt.expected {
				t.Errorf("channel type: got %s, want %s", tt.config.Type, tt.expected)
			}
		})
	}
}

func TestConfigCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.json")

	cfg := DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}

	if !info.Mode().IsRegular() {
		t.Error("config file should be regular file")
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("config file should have 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestDatabaseConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Database.Path == "" {
		t.Error("expected database path to be set")
	}

	if cfg.Database.PragmaPath == "" {
		t.Error("expected pragma path to be set")
	}

	if filepath.Base(cfg.Database.Path) != "mortis.db" {
		t.Errorf("expected database file name mortis.db, got %s", filepath.Base(cfg.Database.Path))
	}
}

func TestServerConfig(t *testing.T) {
	tests := []struct {
		name   string
		config ServerConfig
	}{
		{
			name: "minimal config",
			config: ServerConfig{
				Host: "127.0.0.1",
				Port: 8080,
			},
		},
		{
			name: "with TLS disabled",
			config: ServerConfig{
				Host:        "0.0.0.0",
				Port:        443,
				TLSDisabled: true,
			},
		},
		{
			name: "with TLS enabled",
			config: ServerConfig{
				Host:     "example.com",
				Port:     443,
				CertFile: "/path/to/cert.pem",
				KeyFile:  "/path/to/key.pem",
			},
		},
		{
			name: "with custom timeouts",
			config: ServerConfig{
				Host:         "127.0.0.1",
				Port:         8080,
				ReadTimeout:  60,
				WriteTimeout: 60,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Port <= 0 || tt.config.Port > 65535 {
				t.Errorf("invalid port %d", tt.config.Port)
			}

			if tt.config.Host == "" {
				t.Error("host should not be empty")
			}
		})
	}
}

func TestGatewayConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Gateway.Name == "" {
		t.Error("gateway name should not be empty")
	}

	if cfg.Gateway.AgentTimeout <= 0 {
		t.Errorf("expected positive agent timeout, got %d", cfg.Gateway.AgentTimeout)
	}

	if cfg.Gateway.MaxToolIterations <= 0 {
		t.Errorf("expected positive max tool iterations, got %d", cfg.Gateway.MaxToolIterations)
	}

	if cfg.Gateway.ToolResultSanitizationLimit <= 0 {
		t.Errorf("expected positive sanitization limit, got %d", cfg.Gateway.ToolResultSanitizationLimit)
	}
}
