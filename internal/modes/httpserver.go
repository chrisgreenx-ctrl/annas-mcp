package modes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

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

	// Set up HTTP server with CORS support
	mux := http.NewServeMux()
	mux.Handle("/mcp", corsMiddleware(handler))

	// Add .well-known/mcp-config endpoint for Smithery
	mux.HandleFunc("/.well-known/mcp-config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		configSchema := map[string]interface{}{
			"$schema":     "http://json-schema.org/draft-07/schema#",
			"$id":         "/.well-known/mcp-config",
			"title":       "Anna's Archive MCP Configuration",
			"description": "Configuration for connecting to Anna's Archive MCP server",
			"x-query-style": "dot+bracket",
			"type":        "object",
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
			"required":             []string{"secretKey"},
			"additionalProperties": false,
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(configSchema); err != nil {
			l.Error("Failed to encode config schema", zap.Error(err))
		}
	})

	// Add .well-known/mcp/server-card.json endpoint for server discovery
	mux.HandleFunc("/.well-known/mcp/server-card.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		serverCard := map[string]interface{}{
			"$schema":         "https://modelcontextprotocol.io/schema/server-card.json",
			"version":         "1.0",
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]interface{}{
				"name":    "annas-mcp",
				"title":   "Anna's Archive MCP Server",
				"version": version.GetVersion(),
				"description": "Search and download documents from Anna's Archive",
			},
			"transport": map[string]interface{}{
				"type":     config.TransportType,
				"endpoint": "/mcp",
			},
			"capabilities": map[string]interface{}{
				"tools": "dynamic",
			},
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(serverCard); err != nil {
			l.Error("Failed to encode server card", zap.Error(err))
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
