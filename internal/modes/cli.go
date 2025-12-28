package modes

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/fang"
	"github.com/iosifache/annas-mcp/internal/anna"
	"github.com/iosifache/annas-mcp/internal/logger"
	"github.com/iosifache/annas-mcp/internal/version"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func StartCLI() {
	l := logger.GetLogger()
	defer l.Sync()

	if err := godotenv.Load(); err != nil {
		l.Warn("Error loading .env file", zap.Error(err))
	}

	rootCmd := &cobra.Command{
		Use:   "annas-mcp",
		Short: "Anna's Archive MCP CLI",
		Long:  "A command-line interface for searching and downloading books from Anna's Archive.",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		Version: version.GetVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	searchCmd := &cobra.Command{
		Use:   "search [term]",
		Short: "Search for books",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			searchTerm := args[0]
			l.Info("Search command called", zap.String("searchTerm", searchTerm))

			books, err := anna.FindBook(searchTerm)
			if err != nil {
				l.Error("Search command failed",
					zap.String("searchTerm", searchTerm),
					zap.Error(err),
				)
				return fmt.Errorf("failed to search books: %w", err)
			}

			if len(books) == 0 {
				fmt.Println("No books found.")
				return nil
			}

			for i, book := range books {
				fmt.Printf("Book %d:\n%s\n", i+1, book.String())
				if i < len(books)-1 {
					fmt.Println()
				}
			}

			l.Info("Search command completed successfully",
				zap.String("searchTerm", searchTerm),
				zap.Int("resultsCount", len(books)),
			)

			return nil
		},
	}

	downloadCmd := &cobra.Command{
		Use:   "download [hash]",
		Short: "Get download URL for a book by its MD5 hash",
		Long:  "Get the download URL for a book by its MD5 hash. Requires ANNAS_SECRET_KEY environment variable.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bookHash := args[0]

			l.Info("Download command called",
				zap.String("bookHash", bookHash),
			)

			env, err := GetEnv()
			if err != nil {
				l.Error("Failed to get environment variables", zap.Error(err))
				return fmt.Errorf("failed to get environment: %w", err)
			}

			book := &anna.Book{
				Hash: bookHash,
			}

			url, err := book.GetDownloadURL(env.SecretKey)
			if err != nil {
				l.Error("Download command failed",
					zap.String("bookHash", bookHash),
					zap.Error(err),
				)
				return fmt.Errorf("failed to get download URL: %w", err)
			}

			fmt.Printf("Download URL: %s\n", url)

			l.Info("Download command completed successfully",
				zap.String("bookHash", bookHash),
			)

			return nil
		},
	}

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP server (stdio)",
		Long:  "Start the Model Context Protocol (MCP) server using stdio transport for integration with AI assistants.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Exit CLI mode and start MCP server
			StartMCPServer()
			return nil
		},
	}

	var httpHost string
	var httpPort int
	var httpTransport string

	// Get default port from PORT env var (used by Render, Railway, Heroku, etc.)
	defaultPort := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
			defaultPort = port
		}
	}

	httpCmd := &cobra.Command{
		Use:   "http",
		Short: "Start the MCP server with HTTP transport",
		Long:  "Start the Model Context Protocol (MCP) server using HTTP transport (SSE or Streamable HTTP) for remote access.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := HTTPServerConfig{
				Host:          httpHost,
				Port:          httpPort,
				TransportType: httpTransport,
			}
			return StartHTTPServer(config)
		},
	}

	httpCmd.Flags().StringVar(&httpHost, "host", "0.0.0.0", "Host to bind the HTTP server to")
	httpCmd.Flags().IntVar(&httpPort, "port", defaultPort, "Port to bind the HTTP server to (reads from PORT env var if set)")
	httpCmd.Flags().StringVar(&httpTransport, "transport", "streamable", "Transport type: 'sse' or 'streamable' (recommended)")

	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(httpCmd)

	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version.GetVersion()),
	); err != nil {
		os.Exit(1)
	}
}
