package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/config"
)

func TestResolveServerPort_Default(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv.
	t.Setenv(config.EnvConfigFile, "")
	t.Setenv("MNEMONIC_SERVER_PORT", "")

	port, err := resolveServerPort()
	require.NoError(t, err)
	assert.Equal(t, config.DefaultServerPort, port)
}

func TestResolveServerPort_EnvOverride(t *testing.T) {
	t.Setenv(config.EnvConfigFile, "")
	t.Setenv("MNEMONIC_SERVER_PORT", "9999")

	port, err := resolveServerPort()
	require.NoError(t, err)
	assert.Equal(t, 9999, port)
}

// TestResolveServerPort_ConfigFile verifies that the probe reads a port set
// only in the config file, as the server does.
func TestResolveServerPort_ConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("server:\n  port: 9123\n"), 0o600))
	t.Setenv(config.EnvConfigFile, path)
	t.Setenv("MNEMONIC_SERVER_PORT", "")

	port, err := resolveServerPort()
	require.NoError(t, err)
	assert.Equal(t, 9123, port)
}

func TestResolveServerPort_UnreadableExplicitFile(t *testing.T) {
	t.Setenv(config.EnvConfigFile, filepath.Join(t.TempDir(), "missing.yaml"))

	_, err := resolveServerPort()
	assert.ErrorContains(t, err, "failed to read config file")
}

func TestFormatHealthStatus_ValidJSON(t *testing.T) {
	t.Parallel()

	body := `{"status":"OK","name":"mnemonic"}`
	result := formatHealthStatus([]byte(body))
	assert.Equal(t, "mnemonic: OK", result)
}

func TestFormatHealthStatus_InvalidJSON(t *testing.T) {
	t.Parallel()

	body := "not json"
	result := formatHealthStatus([]byte(body))
	assert.Equal(t, "not json", result)
}

func TestFormatHealthStatus_EmptyStatus(t *testing.T) {
	t.Parallel()

	body := `{"name":"mnemonic"}`
	result := formatHealthStatus([]byte(body))
	// Falls back to raw body because status is empty.
	assert.Equal(t, body, result)
}

func TestCheckHealth_Healthy(t *testing.T) {
	resp := map[string]interface{}{
		"status": "OK",
		"name":   "mnemonic",
	}
	body, err := json.Marshal(resp)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, config.DefaultHealthPath, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	// Extract port from test server address.
	port := extractPort(t, srv.URL)
	t.Setenv("MNEMONIC_SERVER_PORT", port)

	exitCode := checkHealth()
	assert.Equal(t, 0, exitCode)
}

func TestCheckHealth_Unhealthy(t *testing.T) {
	resp := map[string]interface{}{
		"status": "Critical",
		"name":   "mnemonic",
	}
	body, err := json.Marshal(resp)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	t.Setenv("MNEMONIC_SERVER_PORT", port)

	exitCode := checkHealth()
	assert.Equal(t, 1, exitCode)
}

func TestCheckHealth_Unreachable(t *testing.T) {
	// Use a port where nothing is listening.
	t.Setenv("MNEMONIC_SERVER_PORT", "1")

	exitCode := checkHealth()
	assert.Equal(t, 1, exitCode)
}

func TestCheckHealth_BadPort(t *testing.T) {
	t.Setenv("MNEMONIC_SERVER_PORT", "not_a_number")

	// The value does not decode as a port, so the probe fails before any request.
	exitCode := checkHealth()
	assert.Equal(t, 1, exitCode)
}

// extractPort parses "http://127.0.0.1:PORT" and returns "PORT" as a string.
func extractPort(t *testing.T, rawURL string) string {
	t.Helper()
	parts := strings.Split(rawURL, ":")
	require.Len(t, parts, 3, "expected URL with scheme:host:port")
	return parts[2]
}
