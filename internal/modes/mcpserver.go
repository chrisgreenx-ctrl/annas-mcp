package modes

import (
	"context"

	"github.com/iosifache/annas-mcp/internal/anna"
	"github.com/iosifache/annas-mcp/internal/logger"
	"github.com/iosifache/annas-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

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
	}, books, nil
}

func DownloadToolHandler(ctx context.Context, req *mcp.CallToolRequest, params DownloadParams) (*mcp.CallToolResult, any, error) {
	l := logger.GetLogger()

	l.Info("Download command called",
		zap.String("bookHash", params.BookHash),
		zap.String("title", params.Title),
		zap.String("format", params.Format),
	)

	env, err := GetEnv()
	if err != nil {
		l.Error("Failed to get environment variables", zap.Error(err))
		return nil, nil, err
	}
	secretKey := env.SecretKey
	downloadPath := env.DownloadPath

	title := params.Title
	format := params.Format
	book := &anna.Book{
		Hash:   params.BookHash,
		Title:  title,
		Format: format,
	}

	err = book.Download(secretKey, downloadPath)
	if err != nil {
		l.Error("Download command failed",
			zap.String("bookHash", params.BookHash),
			zap.String("downloadPath", downloadPath),
			zap.Error(err),
		)
		return nil, nil, err
	}

	l.Info("Download command completed successfully",
		zap.String("bookHash", params.BookHash),
		zap.String("downloadPath", downloadPath),
	)

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{
			Text: "Book downloaded successfully to path: " + downloadPath,
		}},
	}, nil, nil
}

func StartMCPServer() {
	l := logger.GetLogger()
	defer l.Sync()

	serverVersion := version.GetVersion()
	l.Info("Starting MCP server (stdio)",
		zap.String("name", "annas-mcp"),
		zap.String("version", serverVersion),
	)

	server := createMCPServer()

	l.Info("MCP server started successfully")

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		l.Fatal("MCP server failed", zap.Error(err))
	}
}
