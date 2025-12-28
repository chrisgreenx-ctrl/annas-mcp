package modes

import (
	"errors"
	"os"

	"github.com/iosifache/annas-mcp/internal/logger"
	"go.uber.org/zap"
)

type Env struct {
	SecretKey    string `json:"secret"`
	DownloadPath string `json:"download_path"`
}

func GetEnv() (*Env, error) {
	l := logger.GetLogger()

	secretKey := os.Getenv("ANNAS_SECRET_KEY")
	downloadPath := os.Getenv("ANNAS_DOWNLOAD_PATH")
	if secretKey == "" {
		err := errors.New("ANNAS_SECRET_KEY environment variable must be set")

		l.Error("Environment variables not set",
			zap.String("ANNAS_SECRET_KEY", secretKey),
			zap.Error(err),
		)

		return nil, err
	}

	return &Env{
		SecretKey:    secretKey,
		DownloadPath: downloadPath,
	}, nil
}
