// Package config provides configuration management for Mortis
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the main configuration structure
type Config struct {
	// Server contains server configuration
	Server ServerConfig `json:"server"`
	// Auth contains authentication configuration
	Auth AuthConfig `json:"auth"`
	// Gateway contains gateway configuration
	Gateway GatewayConfig `json:"gateway"`
	// Database contains database configuration
	Database DatabaseConfig `json:"database"`
	// Providers contains provider configurations
	Providers map[string]ProviderConfig `json:"providers"`
	// Session contains session configuration
	Session SessionConfig `json:"session"`
	// Tools contains tools configuration
	Tools ToolsConfig `json:"tools"`
	// Memory contains memory configuration
	Memory *MemoryConfig `json:"memory,omitempty"`
	// Channels contains channel configurations
	Channels map[string]ChannelConfig `json:"channels,omitempty"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	// Host is the server host address
	Host string `json:"host"`
	// Port is the server port
	Port int `json:"port"`
	// TLSDisabled disables TLS
	TLSDisabled bool `json:"tlsDisabled"`
	// CertFile is the TLS certificate file path
	CertFile string `json:"certFile,omitempty"`
	// KeyFile is the TLS key file path
	KeyFile string `json:"keyFile,omitempty"`
	// ReadTimeout is the read timeout in seconds
	ReadTimeout int `json:"readTimeout"`
	// WriteTimeout is the write timeout in seconds
	WriteTimeout int `json:"writeTimeout"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	// Disabled disables authentication
	Disabled bool `json:"disabled"`
	// PasswordHash is the argon2id hashed password
	PasswordHash string `json:"passwordHash,omitempty"`
	// SessionToken is the session token
	SessionToken string `json:"sessionToken,omitempty"`
	// APIKeys are API keys for bearer auth
	APIKeys []string `json:"apiKeys,omitempty"`
	// Passkeys are registered WebAuthn passkeys
	Passkeys []PasskeyConfig `json:"passkeys,omitempty"`
}

// PasskeyConfig contains WebAuthn passkey configuration
type PasskeyConfig struct {
	// ID is the passkey ID
	ID string `json:"id"`
	// CredentialID is the credential ID
	CredentialID string `json:"credentialId,omitempty"`
	// Name is the passkey name
	Name string `json:"name,omitempty"`
	// AddedAt is the timestamp when added
	AddedAt int64 `json:"addedAt,omitempty"`
}

// GatewayConfig contains gateway configuration
type GatewayConfig struct {
	// Name is the gateway name
	Name string `json:"name"`
	// Emoji is the gateway emoji
	Emoji string `json:"emoji,omitempty"`
	// DefaultSessionID is the default session ID
	DefaultSessionID string `json:"defaultSessionId"`
	// DefaultModel is the default model
	DefaultModel string `json:"defaultModel"`
	// DefaultProvider is the default provider
	DefaultProvider string `json:"defaultProvider"`
	// AgentTimeout is the agent timeout in seconds
	AgentTimeout int `json:"agentTimeout"`
	// MaxToolIterations is the maximum tool iterations
	MaxToolIterations int `json:"maxToolIterations"`
	// MessageQueueMode is the message queue mode
	MessageQueueMode string `json:"messageQueueMode"`
	// ToolResultSanitizationLimit is the sanitization limit in bytes
	ToolResultSanitizationLimit int `json:"toolResultSanitizationLimit"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	// Path is the database file path
	Path string `json:"path"`
	// PragmaPath is the pragma path
	PragmaPath string `json:"pragmaPath,omitempty"`
}

// ProviderConfig contains provider configuration
type ProviderConfig struct {
	// Type is the provider type (openai, anthropic, local)
	Type string `json:"type"`
	// APIKey is the API key
	APIKey string `json:"apiKey,omitempty"`
	// BaseURL is the base URL (for custom endpoints)
	BaseURL string `json:"baseUrl,omitempty"`
	// Model is the default model
	Model string `json:"model,omitempty"`
	// Models is the list of available models
	Models []string `json:"models,omitempty"`
	// Enabled is whether the provider is enabled
	Enabled bool `json:"enabled"`
}

// SessionConfig contains session configuration
type SessionConfig struct {
	// DataDir is the sessions data directory
	DataDir string `json:"dataDir"`
	// AutoCompactEnabled enables automatic session compaction
	AutoCompactEnabled bool `json:"autoCompactEnabled"`
	// CompactThreshold is the compaction threshold (0.0-1.0)
	CompactThreshold float64 `json:"compactThreshold"`
	// HistoryLimit is the maximum history size
	HistoryLimit int `json:"historyLimit"`
}

// ToolsConfig contains tools configuration
type ToolsConfig struct {
	// Exec contains execution tool configuration
	Exec ExecToolConfig `json:"exec"`
	// WebFetch contains web fetch tool configuration
	WebFetch WebFetchToolConfig `json:"webFetch,omitempty"`
}

// ExecToolConfig contains execution tool configuration
type ExecToolConfig struct {
	// Enabled enables the execution tool
	Enabled bool `json:"enabled"`
	// Sandbox contains sandbox configuration
	Sandbox SandboxConfig `json:"sandbox"`
}

// SandboxConfig contains sandbox configuration
type SandboxConfig struct {
	// Backend is the sandbox backend (docker, apple, disabled)
	Backend string `json:"backend"`
	// BaseImage is the base image
	BaseImage string `json:"baseImage"`
	// Packages are the packages to install
	Packages []string `json:"packages"`
}

// WebFetchToolConfig contains web fetch tool configuration
type WebFetchToolConfig struct {
	// Enabled enables the web fetch tool
	Enabled bool `json:"enabled"`
	// Timeout is the request timeout in seconds
	Timeout int `json:"timeout"`
	// BlockedDomains are domains to block
	BlockedDomains []string `json:"blockedDomains,omitempty"`
}

// MemoryConfig contains memory configuration
type MemoryConfig struct {
	// Enabled enables memory
	Enabled bool `json:"enabled"`
	// Provider is the memory provider (sqlite, postgres)
	Provider string `json:"provider"`
	// DataDir is the memory data directory
	DataDir string `json:"dataDir"`
	// EmbeddingModel is the embedding model
	EmbeddingModel string `json:"embeddingModel,omitempty"`
	// ChunkSize is the chunk size in tokens
	ChunkSize int `json:"chunkSize"`
	// ChunkOverlap is the chunk overlap in tokens
	ChunkOverlap int `json:"chunkOverlap"`
}

// ChannelConfig contains channel configuration
type ChannelConfig struct {
	// Type is the channel type (telegram, discord, slack)
	Type string `json:"type"`
	// Enabled is whether the channel is enabled
	Enabled bool `json:"enabled"`
	// Token is the channel bot token
	Token string `json:"token,omitempty"`
	// WebhookURL is the webhook URL
	WebhookURL string `json:"webhookUrl,omitempty"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		Server: ServerConfig{
			Host:         "127.0.0.1",
			Port:         13131,
			TLSDisabled:  false,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Auth: AuthConfig{
			Disabled: true,
		},
		Gateway: GatewayConfig{
			Name:                        "Mortis",
			Emoji:                       "🤖",
			DefaultSessionID:            "default",
			DefaultProvider:             "openai",
			DefaultModel:                "gpt-4o",
			AgentTimeout:                600,
			MaxToolIterations:           25,
			MessageQueueMode:            "followup",
			ToolResultSanitizationLimit: 50 * 1024,
		},
		Database: DatabaseConfig{
			Path:       filepath.Join(homeDir, ".mortis", "mortis.db"),
			PragmaPath: filepath.Join(homeDir, ".mortis", "pragmas"),
		},
		Providers: map[string]ProviderConfig{
			"openai": {
				Type:    "openai",
				Model:   "gpt-4o",
				Models:  []string{"gpt-4o", "gpt-4o-mini", "o1-preview"},
				Enabled: true,
			},
		},
		Session: SessionConfig{
			DataDir:            filepath.Join(homeDir, ".mortis", "sessions"),
			AutoCompactEnabled: true,
			CompactThreshold:   0.95,
			HistoryLimit:       200,
		},
		Tools: ToolsConfig{
			Exec: ExecToolConfig{
				Enabled: true,
				Sandbox: SandboxConfig{
					Backend:   "disabled",
					BaseImage: "ubuntu:25.10",
					Packages:  []string{"curl", "jq"},
				},
			},
			WebFetch: WebFetchToolConfig{
				Enabled: true,
				Timeout: 30,
			},
		},
		Memory: &MemoryConfig{
			Enabled:      true,
			Provider:     "sqlite",
			DataDir:      filepath.Join(homeDir, ".mortis", "memory"),
			ChunkSize:    512,
			ChunkOverlap: 64,
		},
	}
}

// Save saves configuration to a file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
