// Package gateway provides the HTTP/WebSocket gateway server for Mortis
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Inokinoki/mortis/pkg/config"
	"github.com/Inokinoki/mortis/pkg/protocol"
	"github.com/Inokinoki/mortis/pkg/provider"
	"github.com/Inokinoki/mortis/pkg/session"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Server is the gateway server
type Server struct {
	config      *config.Config
	router      *gin.Engine
	httpServer  *http.Server
	providerReg *provider.Registry
	sessionMgr  *session.Manager
	upgrader    *websocket.Upgrader
	mu          sync.RWMutex
	connections map[*websocket.Conn]struct{}
	seq         uint64
}

// New creates a new gateway server
func New(cfg *config.Config) *Server {
	// Set up Gin router
	router := gin.Default()
	router.Use(gin.Recovery())

	s := &Server{
		config:      cfg,
		router:      router,
		providerReg: provider.NewRegistry(),
		sessionMgr:  session.NewManager(cfg.Session.DataDir),
		seq:         0,
		connections: make(map[*websocket.Conn]struct{}),
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	s.setupRoutes()

	return s
}

// setupRoutes sets up HTTP routes
func (s *Server) setupRoutes() {
	// Static files
	s.router.Static("/assets", "./web/ui/assets")
	s.router.StaticFile("/", "./web/ui/index.html")

	// WebSocket endpoint
	s.router.GET("/ws", s.handleWebSocket)

	// API routes
	api := s.router.Group("/api")

	// Health check
	api.GET("/health", s.handleHealth)

	// Auth routes
	auth := api.Group("/auth")
	auth.GET("/status", s.handleAuthStatus)
	auth.POST("/setup", s.handleAuthSetup)
	auth.POST("/login", s.handleAuthLogin)
	auth.POST("/logout", s.handleAuthLogout)

	// Chat routes
	chat := api.Group("/chat")
	chat.POST("/send", s.handleChatSend)
	chat.GET("/history", s.handleChatHistory)

	// Session routes
	sessions := api.Group("/session")
	sessions.GET("/list", s.handleSessionList)
	sessions.POST("/create", s.handleSessionCreate)
	sessions.DELETE("/:id", s.handleSessionDelete)

	// Provider routes
	providers := api.Group("/provider")
	providers.GET("/list", s.handleProviderList)
	providers.POST("/configure", s.handleProviderConfigure)
	providers.POST("/test", s.handleProviderTest)

	// Config routes
	configRoutes := api.Group("/config")
	configRoutes.GET("/get", s.handleConfigGet)
	configRoutes.POST("/set", s.handleConfigSet)
	configRoutes.POST("/validate", s.handleConfigValidate)
}

// Start starts the gateway server
func (s *Server) Start(ctx context.Context) error {
	// Register providers
	if err := s.registerProviders(); err != nil {
		return fmt.Errorf("failed to register providers: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	log.Printf("Starting gateway server on %s (tls=%v)", addr, !s.config.Server.TLSDisabled)

	// Start server in goroutine
	errChan := make(chan error, 1)

	go func() {
		if s.config.Server.TLSDisabled {
			errChan <- s.httpServer.ListenAndServe()
		} else {
			errChan <- s.httpServer.ListenAndServeTLS(
				s.config.Server.CertFile,
				s.config.Server.KeyFile,
			)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Shutting down gateway server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		s.httpServer.Shutdown(shutdownCtx)
		return nil
	case err := <-errChan:
		return err
	}
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusOK, protocol.NewResponseErr("", protocol.NewError(protocol.ErrorCodeInvalidRequest, err.Error())))
		return
	}

	// Check authentication
	if !s.isAuthenticated(c) {
		conn.Close()
		c.JSON(http.StatusOK, protocol.NewResponseErr("", protocol.NewError(protocol.ErrorCodeNotPaired, "Not authenticated")))
		return
	}

	// Add to connections
	s.mu.Lock()
	s.connections[conn] = struct{}{}
	s.mu.Unlock()

	// Send connected event
	s.seq++
	s.sendEvent(conn, protocol.EventFrame{
		Type:  protocol.FrameTypeEvent,
		Event: protocol.EventGatewayStart,
		Seq:   &s.seq,
	})

	// Handle messages
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.connections, conn)
		s.mu.Unlock()
	}()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if messageType != websocket.TextMessage {
			continue
		}

		// Parse frame
		var frame protocol.Frame
		if err := json.Unmarshal(data, &frame); err != nil {
			log.Printf("Failed to parse frame: %v", err)
			continue
		}

		// Handle request
		if frame.Type == protocol.FrameTypeRequest && frame.Request != nil {
			s.handleRequest(conn, *frame.Request)
		}
	}
}

// handleRequest handles RPC requests
func (s *Server) handleRequest(conn *websocket.Conn, req protocol.RequestFrame) {
	switch req.Method {
	case protocol.MethodChatSend:
		s.handleChatSendRPC(conn, req)
	case protocol.MethodChatHistory:
		s.handleChatHistoryRPC(conn, req)
	case protocol.MethodSessionList:
		s.handleSessionListRPC(conn, req)
	case protocol.MethodSessionCreate:
		s.handleSessionCreateRPC(conn, req)
	case protocol.MethodSessionDelete:
		s.handleSessionDeleteRPC(conn, req)
	case protocol.MethodProviderList:
		s.handleProviderListRPC(conn, req)
	default:
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnknownMethod, "Unknown method")))
	}
}

// handleHealth handles health check
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": protocol.ProtocolVersion,
	})
}

// handleAuthStatus handles auth status check
func (s *Server) handleAuthStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"authenticated": s.isAuthenticated(c),
	})
}

// handleAuthSetup handles initial setup
func (s *Server) handleAuthSetup(c *gin.Context) {
	// When auth is disabled, setup is already complete
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"configured": true,
	})
}

// handleAuthLogin handles login
func (s *Server) handleAuthLogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "auth_disabled",
	})
}

// handleAuthLogout handles logout
func (s *Server) handleAuthLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// handleChatSend handles chat.send RPC
func (s *Server) handleChatSendRPC(conn *websocket.Conn, req protocol.RequestFrame) {
	var params protocol.ChatSendParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeInvalidRequest, "Invalid params")))
		return
	}

	// Use default session if not specified
	sessionID := params.SessionID
	if sessionID == "" {
		sessionID = s.config.Gateway.DefaultSessionID
	}

	// Check if session exists
	if _, ok := s.sessionMgr.Get(sessionID); !ok {
		// Create default session
		sess, err := s.sessionMgr.Create(
			s.config.Gateway.Name,
			s.config.Gateway.DefaultModel,
			s.config.Gateway.DefaultProvider,
		)
		if err != nil {
			s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnavailable, "Failed to create session")))
			return
		}
		sessionID = sess.ID
	}

	// Create message ID
	messageID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	// Send thinking event
	s.seq++
	thinkingPayload, _ := json.Marshal(protocol.ThinkingEventPayload{
		SessionID: sessionID,
		MessageID: messageID,
	})
	s.sendEvent(conn, protocol.EventFrame{
		Type:    protocol.FrameTypeEvent,
		Event:   protocol.EventChatThinking,
		Seq:     &s.seq,
		Payload: thinkingPayload,
	})

	// Get the session
	sess, ok := s.sessionMgr.Get(sessionID)
	if !ok {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnavailable, "Session not found")))
		return
	}

	// Get the provider
	prov, ok := s.providerReg.Get(sess.Provider)
	if !ok {
		// Try default provider
		prov, ok = s.providerReg.GetDefault()
		if !ok {
			s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnknownProvider, "No provider available")))
			return
		}
	}

	// Get message history
	messages, err := s.sessionMgr.GetMessages(context.Background(), sessionID, 0, "")
	if err != nil {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnavailable, "Failed to get message history")))
		return
	}

	// Convert to provider format
	providerMessages := make([]provider.Message, len(messages)+1)
	for i, msg := range messages {
		providerMessages[i] = provider.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	// Add user message
	providerMessages[len(messages)] = provider.Message{
		Role:    "user",
		Content: params.Message,
	}

	// Build completion request
	compReq := provider.CompletionRequest{
		SessionID:   sessionID,
		MessageID:   messageID,
		Messages:    providerMessages,
		Model:       params.Model,
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	// Default to streaming
	stream := true
	if params.Stream != nil {
		stream = *params.Stream
	}

	if stream {
		// Streaming response
		eventCh, err := prov.Stream(context.Background(), compReq)
		if err != nil {
			s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnavailable, err.Error())))
			return
		}

		fullContent := ""
		for event := range eventCh {
			switch event.Type {
			case provider.StreamEventTypeTextDelta:
				fullContent += event.Content
				s.seq++
				textDeltaPayload, _ := json.Marshal(protocol.TextDeltaEventPayload{
					SessionID: sessionID,
					MessageID: messageID,
					Delta:     event.Content,
					Done:      false,
				})
				s.sendEvent(conn, protocol.EventFrame{
					Type:    protocol.FrameTypeEvent,
					Event:   protocol.EventChatTextDelta,
					Seq:     &s.seq,
					Payload: textDeltaPayload,
				})
			case provider.StreamEventTypeDone:
				finishReason := event.FinishReason
				if finishReason == "" {
					finishReason = "stop"
				}
				s.seq++
				donePayload, _ := json.Marshal(protocol.DoneEventPayload{
					SessionID:    sessionID,
					MessageID:    messageID,
					FinishReason: finishReason,
					TokensUsed:   event.TokensUsed,
				})
				s.sendEvent(conn, protocol.EventFrame{
					Type:    protocol.FrameTypeEvent,
					Event:   protocol.EventChatDone,
					Seq:     &s.seq,
					Payload: donePayload,
				})
			}
		}

		// Save messages to session
		s.sessionMgr.AddMessage(context.Background(), sessionID, protocol.Message{
			ID:        messageID,
			Role:      "user",
			Content:   params.Message,
			Timestamp: time.Now().Unix(),
		})
		s.sessionMgr.AddMessage(context.Background(), sessionID, protocol.Message{
			ID:        messageID + "_resp",
			Role:      "assistant",
			Content:   fullContent,
			Timestamp: time.Now().Unix(),
		})
	} else {
		// Non-streaming response
		resp, err := prov.Complete(context.Background(), compReq)
		if err != nil {
			s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnavailable, err.Error())))
			return
		}

		// Send text delta event
		s.seq++
		textDeltaPayload, _ := json.Marshal(protocol.TextDeltaEventPayload{
			SessionID: sessionID,
			MessageID: messageID,
			Delta:     resp.Content,
			Done:      true,
		})
		s.sendEvent(conn, protocol.EventFrame{
			Type:    protocol.FrameTypeEvent,
			Event:   protocol.EventChatTextDelta,
			Seq:     &s.seq,
			Payload: textDeltaPayload,
		})

		// Send done event
		s.seq++
		donePayload, _ := json.Marshal(protocol.DoneEventPayload{
			SessionID:    sessionID,
			MessageID:    messageID,
			FinishReason: resp.FinishReason,
			TokensUsed:   resp.TokensUsed,
		})
		s.sendEvent(conn, protocol.EventFrame{
			Type:    protocol.FrameTypeEvent,
			Event:   protocol.EventChatDone,
			Seq:     &s.seq,
			Payload: donePayload,
		})

		// Save messages to session
		s.sessionMgr.AddMessage(context.Background(), sessionID, protocol.Message{
			ID:        messageID,
			Role:      "user",
			Content:   params.Message,
			Timestamp: time.Now().Unix(),
		})
		s.sessionMgr.AddMessage(context.Background(), sessionID, protocol.Message{
			ID:        messageID + "_resp",
			Role:      "assistant",
			Content:   resp.Content,
			Timestamp: time.Now().Unix(),
		})
	}

	// Send RPC response
	respPayload, _ := json.Marshal(protocol.ChatSendPayload{
		SessionID: sessionID,
		MessageID: messageID,
	})
	s.sendResponse(conn, req.ID, protocol.NewResponseOK(req.ID, respPayload))
}

// handleChatHistoryRPC handles chat.history RPC
func (s *Server) handleChatHistoryRPC(conn *websocket.Conn, req protocol.RequestFrame) {
	var params protocol.ChatHistoryParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeInvalidRequest, "Invalid params")))
		return
	}

	messages, err := s.sessionMgr.GetMessages(context.Background(), params.SessionID, 0, "")
	if err != nil {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnavailable, "Failed to get messages")))
		return
	}

	respPayload, _ := json.Marshal(protocol.ChatHistoryPayload{
		Messages: messages,
		HasMore:  false,
	})
	s.sendResponse(conn, req.ID, protocol.NewResponseOK(req.ID, respPayload))
}

// handleSessionListRPC handles session.list RPC
func (s *Server) handleSessionListRPC(conn *websocket.Conn, req protocol.RequestFrame) {
	sessions := s.sessionMgr.List()
	respPayload, _ := json.Marshal(sessions)
	s.sendResponse(conn, req.ID, protocol.NewResponseOK(req.ID, respPayload))
}

// handleProviderListRPC handles provider.list RPC
func (s *Server) handleProviderListRPC(conn *websocket.Conn, req protocol.RequestFrame) {
	providers := s.providerReg.List()

	result := make([]gin.H, 0, len(providers))
	for _, p := range providers {
		info, err := p.Info(context.Background())
		if err != nil {
			continue
		}
		result = append(result, gin.H{
			"id":        info.ID,
			"name":      info.Name,
			"type":      info.Type,
			"available": info.Available,
			"models":    info.Models,
			"features":  info.Features,
		})
	}

	respPayload, _ := json.Marshal(result)
	s.sendResponse(conn, req.ID, protocol.NewResponseOK(req.ID, respPayload))
}

// handleSessionCreateRPC handles session.create RPC
func (s *Server) handleSessionCreateRPC(conn *websocket.Conn, req protocol.RequestFrame) {
	var params struct {
		Name     string `json:"name"`
		Model    string `json:"model,omitempty"`
		Provider string `json:"provider,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeInvalidRequest, "Invalid params")))
		return
	}

	model := params.Model
	if model == "" {
		model = s.config.Gateway.DefaultModel
	}

	provider := params.Provider
	if provider == "" {
		provider = s.config.Gateway.DefaultProvider
	}

	sess, err := s.sessionMgr.Create(params.Name, model, provider)
	if err != nil {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnavailable, "Failed to create session")))
		return
	}

	respPayload, _ := json.Marshal(sess)
	s.sendResponse(conn, req.ID, protocol.NewResponseOK(req.ID, respPayload))
}

// handleSessionDeleteRPC handles session.delete RPC
func (s *Server) handleSessionDeleteRPC(conn *websocket.Conn, req protocol.RequestFrame) {
	var params struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeInvalidRequest, "Invalid params")))
		return
	}

	if err := s.sessionMgr.Delete(params.ID); err != nil {
		s.sendResponse(conn, req.ID, protocol.NewResponseErr(req.ID, protocol.NewError(protocol.ErrorCodeUnavailable, "Failed to delete session")))
		return
	}

	respPayload, _ := json.Marshal(gin.H{"status": "deleted"})
	s.sendResponse(conn, req.ID, protocol.NewResponseOK(req.ID, respPayload))
}

// sendResponse sends a response frame
func (s *Server) sendResponse(conn *websocket.Conn, id string, resp protocol.ResponseFrame) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

// sendEvent sends an event frame
func (s *Server) sendEvent(conn *websocket.Conn, event protocol.EventFrame) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send event: %v", err)
	}
}

// isAuthenticated checks if request is authenticated
func (s *Server) isAuthenticated(c *gin.Context) bool {
	if s.config.Auth.Disabled {
		return true
	}

	// Check localhost
	host := c.Request.Host
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// Check for API key in header
	apiKey := c.GetHeader("X-API-Key")
	for _, key := range s.config.Auth.APIKeys {
		if key == apiKey {
			return true
		}
	}

	// Check for session cookie
	if cookie, err := c.Cookie("mortis_session"); err == nil {
		if cookie == s.config.Auth.SessionToken {
			return true
		}
	}

	return false
}

// registerProviders registers LLM providers
func (s *Server) registerProviders() error {
	for id, cfg := range s.config.Providers {
		if !cfg.Enabled {
			continue
		}

		var p provider.LLM
		var err error

		switch cfg.Type {
		case "openai":
			p = provider.NewOpenAI(cfg)
		case "anthropic":
			p = provider.NewAnthropic(cfg)
		case "local":
			p = provider.NewLocalLLM(cfg)
		default:
			log.Printf("Unknown provider type: id=%s type=%s", id, cfg.Type)
			continue
		}

		if err != nil {
			log.Printf("Failed to create provider id=%s: %v", id, err)
			continue
		}

		s.providerReg.Register(id, p)

		// Set as default if configured
		if s.config.Gateway.DefaultProvider == id {
			s.providerReg.SetDefault(id)
		}
	}

	return nil
}

// HTTP handlers for REST API (placeholder implementations)

func (s *Server) handleChatSend(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "use_websocket"})
}

func (s *Server) handleChatHistory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "use_websocket"})
}

func (s *Server) handleSessionList(c *gin.Context) {
	sessions := s.sessionMgr.List()
	c.JSON(http.StatusOK, sessions)
}

func (s *Server) handleSessionCreate(c *gin.Context) {
	var req struct {
		Name     string `json:"name"`
		Model    string `json:"model,omitempty"`
		Provider string `json:"provider,omitempty"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	model := req.Model
	if model == "" {
		model = s.config.Gateway.DefaultModel
	}

	provider := req.Provider
	if provider == "" {
		provider = s.config.Gateway.DefaultProvider
	}

	sess, err := s.sessionMgr.Create(req.Name, model, provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sess)
}

func (s *Server) handleSessionDelete(c *gin.Context) {
	id := c.Param("id")
	if err := s.sessionMgr.Delete(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (s *Server) handleProviderList(c *gin.Context) {
	providers := s.providerReg.List()

	result := make([]gin.H, 0, len(providers))
	for _, p := range providers {
		info, err := p.Info(context.Background())
		if err != nil {
			continue
		}
		result = append(result, gin.H{
			"id":        info.ID,
			"name":      info.Name,
			"type":      info.Type,
			"available": info.Available,
			"models":    info.Models,
			"features":  info.Features,
		})
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleProviderConfigure(c *gin.Context) {
	var req struct {
		ID       string                `json:"id"`
		Provider config.ProviderConfig `json:"provider"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update provider config
	s.config.Providers[req.ID] = req.Provider

	// Re-register providers
	if err := s.registerProviders(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "configured"})
}

func (s *Server) handleProviderTest(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider, ok := s.providerReg.Get(req.ID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Test provider by calling Info
	info, err := provider.Info(context.Background())
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "error",
			"error":     err.Error(),
			"available": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"available": info.Available,
		"info":      info,
	})
}

func (s *Server) handleConfigGet(c *gin.Context) {
	c.JSON(http.StatusOK, s.config)
}

func (s *Server) handleConfigSet(c *gin.Context) {
	var cfg config.Config
	if err := c.BindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update config
	s.config = &cfg

	// Re-register providers
	if err := s.registerProviders(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (s *Server) handleConfigValidate(c *gin.Context) {
	var cfg config.Config
	if err := c.BindJSON(&cfg); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	// Basic validation
	valid := true
	errors := []string{}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		valid = false
		errors = append(errors, "Invalid port number")
	}

	if cfg.Gateway.DefaultProvider == "" {
		valid = false
		errors = append(errors, "Default provider is required")
	}

	if cfg.Gateway.DefaultModel == "" {
		valid = false
		errors = append(errors, "Default model is required")
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":  valid,
		"errors": errors,
	})
}
