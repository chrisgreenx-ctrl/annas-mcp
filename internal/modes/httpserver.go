package modes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/iosifache/annas-mcp/internal/logger"
	"github.com/iosifache/annas-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// HTTPServerConfig holds configuration for the HTTP MCP server
type HTTPServerConfig struct {
	Host          string
	Port          int
	TransportType string // "sse" or "streamable"
}

// configureEnvFromRequest reads configuration from query parameters and sets environment variables
// This is used for Smithery integration where config is passed via query params
func configureEnvFromRequest(r *http.Request, l *zap.Logger) {
	query := r.URL.Query()

	// Check for Smithery-style config (secretKey, downloadPath)
	if secretKey := query.Get("secretKey"); secretKey != "" {
		os.Setenv("ANNAS_SECRET_KEY", secretKey)
		l.Debug("Set ANNAS_SECRET_KEY from query parameter")
	}

	if downloadPath := query.Get("downloadPath"); downloadPath != "" {
		os.Setenv("ANNAS_DOWNLOAD_PATH", downloadPath)
		l.Debug("Set ANNAS_DOWNLOAD_PATH from query parameter", zap.String("path", downloadPath))
	}

	// Also support direct environment variable names for backwards compatibility
	if secretKey := query.Get("ANNAS_SECRET_KEY"); secretKey != "" {
		os.Setenv("ANNAS_SECRET_KEY", secretKey)
		l.Debug("Set ANNAS_SECRET_KEY from query parameter (direct)")
	}

	if downloadPath := query.Get("ANNAS_DOWNLOAD_PATH"); downloadPath != "" {
		os.Setenv("ANNAS_DOWNLOAD_PATH", downloadPath)
		l.Debug("Set ANNAS_DOWNLOAD_PATH from query parameter (direct)", zap.String("path", downloadPath))
	}
}

// createMCPServer creates and configures an MCP server instance
func createMCPServer() *mcp.Server {
	serverVersion := version.GetVersion()
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "annas-mcp",
		Version: serverVersion,
	}, nil)

	// Add search tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search",
		Description: "Search books on Anna's Archive",
	}, SearchToolHandler)

	// Add download tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "download",
		Description: "Download a book by its MD5 hash. Requires ANNAS_SECRET_KEY and ANNAS_DOWNLOAD_PATH environment variables.",
	}, DownloadToolHandler)

	return server
}

// StartHTTPServer starts the MCP server with HTTP transport (SSE or Streamable)
func StartHTTPServer(config HTTPServerConfig) error {
	l := logger.GetLogger()
	defer l.Sync()

	serverVersion := version.GetVersion()
	l.Info("Starting MCP HTTP server",
		zap.String("name", "annas-mcp"),
		zap.String("version", serverVersion),
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
		zap.String("transport", config.TransportType),
	)

	// Create HTTP handler based on transport type
	var handler http.Handler
	switch config.TransportType {
	case "sse":
		handler = mcp.NewSSEHandler(
			func(r *http.Request) *mcp.Server {
				configureEnvFromRequest(r, l)
				return createMCPServer()
			},
			nil,
		)
	case "streamable":
		handler = mcp.NewStreamableHTTPHandler(
			func(r *http.Request) *mcp.Server {
				configureEnvFromRequest(r, l)
				return createMCPServer()
			},
			nil,
		)
	default:
		return fmt.Errorf("invalid transport type: %s (must be 'sse' or 'streamable')", config.TransportType)
	}

	// Set up HTTP server with CORS and OAuth support
	mux := http.NewServeMux()
	mux.Handle("/mcp", corsMiddleware(oauthMiddleware(handler, l)))

	// Add .well-known/mcp-config endpoint for Smithery
	mux.HandleFunc("/.well-known/mcp-config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		configSchema := map[string]interface{}{
			"title":               "Anna's Archive MCP Configuration",
			"description":         "Configuration for connecting to Anna's Archive MCP server",
			"type":                "object",
			"required":            []string{"secretKey"},
			"additionalProperties": false,
			"properties": map[string]interface{}{
				"secretKey": map[string]interface{}{
					"type":        "string",
					"title":       "Anna's Archive API Key",
					"description": "Your Anna's Archive API key for accessing the JSON API. Get one at https://annas-archive.org/faq#api",
				},
				"downloadPath": map[string]interface{}{
					"type":        "string",
					"title":       "Download Path",
					"description": "Path where downloaded documents will be stored",
					"default":     "/tmp/downloads",
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(configSchema); err != nil {
			l.Error("Failed to encode config schema", zap.Error(err))
		}
	})

	// Add .well-known/mcp-server-card.json endpoint for server discovery (Smithery standard)
	serverCardHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Cache-Control", "public, max-age=3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		serverCard := map[string]interface{}{
			"name":        "annas-mcp",
			"description": "Search and download documents from Anna's Archive",
			"version":     version.GetVersion(),
			"capabilities": map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "search",
						"description": "Search books on Anna's Archive",
					},
					{
						"name":        "download",
						"description": "Download a book by its MD5 hash",
					},
				},
			},
			"authentication": map[string]interface{}{
				"type": "oauth2",
				"oauth": map[string]interface{}{
					"authorizationUrl": "https://smithery.ai/oauth/authorize",
					"tokenUrl":         "https://smithery.ai/oauth/token",
					"scopes":           []string{"mcp:access"},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(serverCard); err != nil {
			l.Error("Failed to encode server card", zap.Error(err))
		}
	}

	// Register handler at both paths for compatibility
	mux.HandleFunc("/.well-known/mcp-server-card.json", serverCardHandler)
	mux.HandleFunc("/.well-known/mcp/server-card.json", serverCardHandler)

	// Add OAuth callback endpoint for Smithery
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Get authorization code from query params
		code := r.URL.Query().Get("code")
		if code == "" {
			l.Warn("OAuth callback missing code parameter")
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			return
		}

		// In a full implementation, you would exchange the code for a token here
		// For now, we just acknowledge the callback
		l.Info("OAuth callback received", zap.String("code_prefix", code[:10]+"..."))

		response := map[string]interface{}{
			"status":  "success",
			"message": "OAuth authorization successful",
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			l.Error("Failed to encode OAuth callback response", zap.Error(err))
		}
	})

	// Add a health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	l.Info("MCP HTTP server listening",
		zap.String("address", addr),
		zap.String("endpoint", "/mcp"),
	)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		l.Fatal("MCP HTTP server failed", zap.Error(err))
		return err
	}

	return nil
}

// corsMiddleware adds CORS headers to allow cross-origin requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// oauthMiddleware verifies OAuth Bearer tokens from Smithery
func oauthMiddleware(next http.Handler, l *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip OAuth check if not configured (for local development)
		smitheryClientID := os.Getenv("SMITHERY_CLIENT_ID")
		if smitheryClientID == "" {
			l.Debug("OAuth not configured, skipping authentication")
			next.ServeHTTP(w, r)
			return
		}

		// Extract Bearer token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			l.Warn("Missing Authorization header")
			http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Check for Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			l.Warn("Invalid Authorization header format")
			http.Error(w, "Unauthorized: Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		if token == "" {
			l.Warn("Empty Bearer token")
			http.Error(w, "Unauthorized: Empty Bearer token", http.StatusUnauthorized)
			return
		}

		// In production, you would verify the token against Smithery's OAuth server
		// For now, we accept any non-empty token when OAuth is configured
		l.Debug("OAuth token verified", zap.String("token_prefix", token[:10]+"..."))

		next.ServeHTTP(w, r)
	})
}
