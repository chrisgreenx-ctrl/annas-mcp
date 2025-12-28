package modes

import (
	"errors"
	"net/http"
	"os"

	"github.com/iosifache/annas-mcp/internal/logger"
	"go.uber.org/zap"
)

type Env struct {
	SecretKey    string `json:"secret"`
	DownloadPath string `json:"download_path"`
}

// LoadEnv resolves the configuration from multiple sources in order of priority:
// 1. Query Parameters (if req is provided)
// 2. Standard Environment Variables (ANNAS_SECRET_KEY, ANNAS_DOWNLOAD_PATH)
// 3. Smithery-style Environment Variables (secretKey, downloadPath)
// 4. Generic Environment Variable (SECRET_KEY)
func LoadEnv(req *http.Request) (*Env, error) {
	l := logger.GetLogger()

	var secretKey string
	var downloadPath string

	// 1. Check Query Parameters (if request is provided)
	if req != nil {
		query := req.URL.Query()
		if val := query.Get("secretKey"); val != "" {
			secretKey = val
		} else if val := query.Get("ANNAS_SECRET_KEY"); val != "" {
			secretKey = val
		}

		if val := query.Get("downloadPath"); val != "" {
			downloadPath = val
		} else if val := query.Get("ANNAS_DOWNLOAD_PATH"); val != "" {
			downloadPath = val
		}
	}

	// 2. Check Standard Environment Variables (if not found in query)
	if secretKey == "" {
		secretKey = os.Getenv("ANNAS_SECRET_KEY")
	}
	if downloadPath == "" {
		downloadPath = os.Getenv("ANNAS_DOWNLOAD_PATH")
	}

	// 3. Check Smithery-style Environment Variables (if not found yet)
	if secretKey == "" {
		secretKey = os.Getenv("secretKey")
	}
	if downloadPath == "" {
		downloadPath = os.Getenv("downloadPath")
	}

	// 4. Check Generic Environment Variable (if not found yet)
	if secretKey == "" {
		secretKey = os.Getenv("SECRET_KEY")
	}

	// Validate required fields
	if secretKey == "" {
		err := errors.New("secretKey must be set via query param, ANNAS_SECRET_KEY, SECRET_KEY, or secretKey env var")
		l.Error("Environment variables not set", zap.Error(err))
		return nil, err
	}

	// Set default download path if not provided
	if downloadPath == "" {
		downloadPath = "/tmp/downloads"
	}

	return &Env{
		SecretKey:    secretKey,
		DownloadPath: downloadPath,
	}, nil
}

// GetEnv is a wrapper around LoadEnv(nil) for backwards compatibility and CLI usage
func GetEnv() (*Env, error) {
	return LoadEnv(nil)
}
