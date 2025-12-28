package modes

import (
	"context"
	"fmt"

	"github.com/iosifache/annas-mcp/internal/anna"
	"github.com/iosifache/annas-mcp/internal/logger"
	"github.com/iosifache/annas-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// SearchToolHandler performs a search on Anna's Archive.
// It does not require any specific environment configuration.
func SearchToolHandler(ctx context.Context, req *mcp.CallToolRequest, params SearchParams) (*mcp.CallToolResult, any, error) {
	l := logger.GetLogger()

	l.Info("Search command called",
		zap.String("searchTerm", params.SearchTerm),
	)

	books, err := anna.FindBook(params.SearchTerm)
	if err != nil {
		l.Error("Search command failed",
			zap.String("searchTerm", params.SearchTerm),
			zap.Error(err),
		)
		return nil, nil, err
	}

	bookList := ""
	for _, book := range books {
		bookList += book.String() + "\n\n"
	}

	l.Info("Search command completed successfully",
		zap.String("searchTerm", params.SearchTerm),
		zap.Int("resultsCount", len(books)),
	)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: bookList}},
	}, map[string]interface{}{"books": books}, nil
}

// NewDownloadToolHandler creates a handler for the download tool that uses the provided environment.
func NewDownloadToolHandler(env *Env) func(context.Context, *mcp.CallToolRequest, DownloadParams) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, params DownloadParams) (*mcp.CallToolResult, any, error) {
		l := logger.GetLogger()

		l.Info("Download command called",
			zap.String("bookHash", params.BookHash),
			zap.String("title", params.Title),
			zap.String("format", params.Format),
		)

		// Use the injected environment instead of global GetEnv()
		secretKey := env.SecretKey

		if secretKey == "" {
			err := fmt.Errorf("secret key is not configured. Please set ANNAS_SECRET_KEY, secretKey, or pass it via query parameters")
			l.Error("Download command failed", zap.Error(err))
			return nil, nil, err
		}

		title := params.Title
		format := params.Format
		book := &anna.Book{
			Hash:   params.BookHash,
			Title:  title,
			Format: format,
		}

		url, err := book.GetDownloadURL(secretKey)
		if err != nil {
			l.Error("Download command failed",
				zap.String("bookHash", params.BookHash),
				zap.Error(err),
			)
			return nil, nil, err
		}

		l.Info("Download command completed successfully",
			zap.String("bookHash", params.BookHash),
		)

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("[%s](%s)", title, url),
			}},
		}, nil, nil
	}
}

// DownloadToolHandler is the legacy handler that uses global env.
// Kept for CLI usage or backward compatibility if needed, but CLI should preferably use NewDownloadToolHandler too if possible.
// However, since CLI "download" command logic is inline in cli.go, this might only be used if someone calls it directly.
// For MCP server, we should use NewDownloadToolHandler.
func DownloadToolHandler(ctx context.Context, req *mcp.CallToolRequest, params DownloadParams) (*mcp.CallToolResult, any, error) {
	// Fallback to global env
	env, err := GetEnv()
	if err != nil {
		return nil, nil, err
	}
	return NewDownloadToolHandler(env)(ctx, req, params)
}


// createMCPServer creates and configures an MCP server instance using the provided environment.
func createMCPServer(env *Env) *mcp.Server {
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
		Description: "Download a book by its MD5 hash. Requires ANNAS_SECRET_KEY/secretKey environment variable.",
	}, NewDownloadToolHandler(env))

	return server
}

// StartMCPServer starts the MCP server in stdio mode.
func StartMCPServer() {
	l := logger.GetLogger()
	defer l.Sync()

	serverVersion := version.GetVersion()
	l.Info("Starting MCP server (stdio)",
		zap.String("name", "annas-mcp"),
		zap.String("version", serverVersion),
	)

	// Load environment from OS (stdio mode doesn't have HTTP request)
	env, err := GetEnv()
	if err != nil {
		// Log error but proceed to allow search tool to work
		l.Warn("Failed to load environment variables, download tool may not work", zap.Error(err))
		env = &Env{}
	}

	server := createMCPServer(env)

	l.Info("MCP server started successfully")

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		l.Fatal("MCP server failed", zap.Error(err))
	}
}
