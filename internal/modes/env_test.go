package modes

import (
	"net/http"
	"os"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	// Test case 1: Priority 1 - Query Parameters
	t.Run("Priority 1: Query Parameters", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com?secretKey=querySecret&downloadPath=queryPath", nil)
		env, err := LoadEnv(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if env.SecretKey != "querySecret" {
			t.Errorf("Expected SecretKey 'querySecret', got '%s'", env.SecretKey)
		}
		if env.DownloadPath != "queryPath" {
			t.Errorf("Expected DownloadPath 'queryPath', got '%s'", env.DownloadPath)
		}
	})

	// Test case 2: Priority 2 - Standard Env Vars
	t.Run("Priority 2: Standard Env Vars", func(t *testing.T) {
		os.Setenv("ANNAS_SECRET_KEY", "stdSecret")
		os.Setenv("ANNAS_DOWNLOAD_PATH", "stdPath")
		defer os.Unsetenv("ANNAS_SECRET_KEY")
		defer os.Unsetenv("ANNAS_DOWNLOAD_PATH")

		// Ensure lower priority vars are not set
		os.Unsetenv("secretKey")
		os.Unsetenv("downloadPath")

		req, _ := http.NewRequest("GET", "http://example.com", nil) // No query params
		env, err := LoadEnv(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if env.SecretKey != "stdSecret" {
			t.Errorf("Expected SecretKey 'stdSecret', got '%s'", env.SecretKey)
		}
		if env.DownloadPath != "stdPath" {
			t.Errorf("Expected DownloadPath 'stdPath', got '%s'", env.DownloadPath)
		}
	})

	// Test case 3: Priority 3 - Smithery Env Vars
	t.Run("Priority 3: Smithery Env Vars", func(t *testing.T) {
		os.Setenv("secretKey", "smitherySecret")
		os.Setenv("downloadPath", "smitheryPath")
		defer os.Unsetenv("secretKey")
		defer os.Unsetenv("downloadPath")

		// Ensure higher priority vars are not set
		os.Unsetenv("ANNAS_SECRET_KEY")
		os.Unsetenv("ANNAS_DOWNLOAD_PATH")

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		env, err := LoadEnv(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if env.SecretKey != "smitherySecret" {
			t.Errorf("Expected SecretKey 'smitherySecret', got '%s'", env.SecretKey)
		}
		if env.DownloadPath != "smitheryPath" {
			t.Errorf("Expected DownloadPath 'smitheryPath', got '%s'", env.DownloadPath)
		}
	})

	// Test case 4: Priority Order
	t.Run("Priority Order", func(t *testing.T) {
		// Set all sources
		os.Setenv("ANNAS_SECRET_KEY", "stdSecret")
		os.Setenv("secretKey", "smitherySecret")
		defer os.Unsetenv("ANNAS_SECRET_KEY")
		defer os.Unsetenv("secretKey")

		// Request with query param
		req, _ := http.NewRequest("GET", "http://example.com?secretKey=querySecret", nil)
		env, err := LoadEnv(req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		// Query should win
		if env.SecretKey != "querySecret" {
			t.Errorf("Expected SecretKey 'querySecret', got '%s'", env.SecretKey)
		}

		// Request without query param
		req2, _ := http.NewRequest("GET", "http://example.com", nil)
		env2, err := LoadEnv(req2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		// Standard Env should win over Smithery Env
		if env2.SecretKey != "stdSecret" {
			t.Errorf("Expected SecretKey 'stdSecret', got '%s'", env2.SecretKey)
		}
	})

	// Test case 5: Missing Secret Key
	t.Run("Missing Secret Key", func(t *testing.T) {
		os.Unsetenv("ANNAS_SECRET_KEY")
		os.Unsetenv("secretKey")
		// unset others just in case
		os.Unsetenv("ANNAS_DOWNLOAD_PATH")
		os.Unsetenv("downloadPath")

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		_, err := LoadEnv(req)
		if err == nil {
			t.Error("Expected error for missing secret key, got nil")
		}
	})
}
