---
entity_name: Cobra Configuration Pattern
entity_type: CLI Pattern
language: go
domain: cli
description: Configuration management pattern with explicit config injection, environment variable overrides, and config precedence for Cobra CLIs
tags:
  - Cobra
  - CLI
  - configuration
  - Go
  - environment-variables
---

# Cobra Configuration Pattern

This pattern demonstrates configuration management for Cobra CLIs with explicit config passing, environment variable overrides, and clear precedence rules.

## Configuration Structure

```go
// internal/config/config.go
package config

import (
    "fmt"
    "os"
    "path/filepath"
)

// Config holds all configuration for the CLI
type Config struct {
    API APIConfig `yaml:"api"`
}

// APIConfig holds API client configuration
type APIConfig struct {
    BaseURL string `yaml:"base_url"`
    Timeout string `yaml:"timeout"`
}

var globalConfig Config

// Initialize loads configuration from file and environment
func Initialize() error {
    // Load from config file
    configPath := getConfigPath()
    if err := loadConfigFile(configPath, &globalConfig); err != nil {
        // Config file is optional, use defaults
        globalConfig = defaultConfig()
    }

    // Apply environment variable overrides
    applyEnvOverrides(&globalConfig)

    return nil
}

// GetConfig returns the global configuration
func GetConfig() Config {
    return globalConfig
}

func defaultConfig() Config {
    return Config{
        API: APIConfig{
            BaseURL: "https://api.example.com",
            Timeout: "30s",
        },
    }
}
```

## Config File Loading

```go
// internal/config/config.go (continued)

import (
    "gopkg.in/yaml.v3"
    "io"
)

func getConfigPath() string {
    // Check for custom config path in environment
    if path := os.Getenv("MYTOOL_CONFIG"); path != "" {
        return path
    }

    // Default to ~/.mytool/config.yaml
    home, err := os.UserHomeDir()
    if err != nil {
        return ""
    }
    return filepath.Join(home, ".mytool", "config.yaml")
}

func loadConfigFile(path string, cfg *Config) error {
    if path == "" {
        return fmt.Errorf("no config path provided")
    }

    file, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("failed to open config file: %w", err)
    }
    defer file.Close()

    data, err := io.ReadAll(file)
    if err != nil {
        return fmt.Errorf("failed to read config file: %w", err)
    }

    if err := yaml.Unmarshal(data, cfg); err != nil {
        return fmt.Errorf("failed to parse config file: %w", err)
    }

    return nil
}
```

## Environment Variable Overrides

```go
// internal/config/config.go (continued)

func applyEnvOverrides(cfg *Config) {
    // Override API base URL
    if baseURL := os.Getenv("MYTOOL_API_BASE_URL"); baseURL != "" {
        cfg.API.BaseURL = baseURL
    }

    // Override API timeout
    if timeout := os.Getenv("MYTOOL_API_TIMEOUT"); timeout != "" {
        cfg.API.Timeout = timeout
    }
}
```

## Config Precedence

Configuration values are applied in this order (last wins):

1. **Default values** - Hardcoded defaults in `defaultConfig()`
2. **Config file** - `~/.mytool/config.yaml` or `$MYTOOL_CONFIG`
3. **Environment variables** - `MYTOOL_*` variables
4. **Command flags** - Flags passed to specific commands

Example:
```bash
# Default timeout is 30s
# Config file sets timeout to 60s
# Environment variable overrides to 90s
export MYTOOL_API_TIMEOUT=90s
# Command flag would override everything (if implemented)
mytool command --timeout 120s
```

## Injecting Config to Commands

All domain commands receive config explicitly:

```go
// cmd/main/main.go
package main

import (
    "log"
    "yourproject/internal/cli/root"
    "yourproject/internal/cli/user"
    "yourproject/internal/cli/company"
    "yourproject/internal/config"
)

func main() {
    // Initialize config first
    if err := config.Initialize(); err != nil {
        log.Fatalln("failed to initialize config:", err)
    }

    // Get config
    cfg := config.GetConfig()

    // Pass config to all domain commands
    userCmd, err := user.Command(&cfg)
    if err != nil {
        log.Fatalln("failed to create user command:", err)
    }

    companyCmd, err := company.Command(&cfg)
    if err != nil {
        log.Fatalln("failed to create company command:", err)
    }

    // Initialize root with all commands
    if err := root.Initialize(userCmd, companyCmd); err != nil {
        log.Fatalln("failed to initialize root:", err)
    }

    root.Execute()
}
```

## Using Config in Commands

```go
// internal/cli/user/add.go
package user

import (
    "context"
    "fmt"
    "time"
    "yourproject/internal/api"
    "yourproject/internal/config"
    "github.com/spf13/cobra"
)

func AddCommand(cfg *config.Config) (*cobra.Command, error) {
    cmd := &cobra.Command{
        Use:   "add",
        Short: "Add a user",
        RunE: func(command *cobra.Command, args []string) error {
            return runUserAdd(command, cfg)
        },
    }
    return cmd, nil
}

func runUserAdd(cmd *cobra.Command, cfg *config.Config) error {
    // Check if API configuration is available
    if cfg.API.BaseURL == "" {
        return fmt.Errorf("API base URL not configured. Please run 'mytool config set --base-url <url>' first")
    }

    // Parse timeout
    timeout, err := time.ParseDuration(cfg.API.Timeout)
    if err != nil {
        return fmt.Errorf("invalid timeout duration: %w", err)
    }

    // Create API client from config
    skipAuth, _ := cmd.Flags().GetBool("skip-auth")
    apiClient := api.NewClient(api.Config{
        BaseURL:  cfg.API.BaseURL,
        Timeout:  timeout,
        SkipAuth: skipAuth,
    })

    // Use client...
    ctx := context.Background()
    return apiClient.CreateUser(ctx, userData)
}
```

## Config Command for Management

```go
// internal/cli/config/config.go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "yourproject/internal/config"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

func Command(cfg *config.Config) (*cobra.Command, error) {
    cmd := &cobra.Command{
        Use:   "config",
        Short: "Manage CLI configuration",
    }

    cmd.AddCommand(setCommand(cfg))
    cmd.AddCommand(getCommand(cfg))
    cmd.AddCommand(showCommand(cfg))

    return cmd, nil
}

func setCommand(cfg *config.Config) *cobra.Command {
    var baseURL string
    var timeout string

    cmd := &cobra.Command{
        Use:   "set",
        Short: "Set configuration values",
        RunE: func(cmd *cobra.Command, args []string) error {
            if baseURL != "" {
                cfg.API.BaseURL = baseURL
            }
            if timeout != "" {
                cfg.API.Timeout = timeout
            }
            return saveConfig(cfg)
        },
    }

    cmd.Flags().StringVar(&baseURL, "base-url", "", "API base URL")
    cmd.Flags().StringVar(&timeout, "timeout", "", "Request timeout (e.g., '30s', '5m')")

    return cmd
}

func showCommand(cfg *config.Config) *cobra.Command {
    return &cobra.Command{
        Use:   "show",
        Short: "Show current configuration",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("Current Configuration:")
            fmt.Printf("  API Base URL: %s\n", cfg.API.BaseURL)
            fmt.Printf("  API Timeout:  %s\n", cfg.API.Timeout)
            fmt.Println("\nConfiguration sources:")
            fmt.Println("  Config file: ~/.mytool/config.yaml")
            fmt.Println("  Environment: MYTOOL_API_BASE_URL, MYTOOL_API_TIMEOUT")
            return nil
        },
    }
}

func saveConfig(cfg *config.Config) error {
    configPath := getConfigPath()

    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }

    // Marshal config to YAML
    data, err := yaml.Marshal(cfg)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    // Write to file
    if err := os.WriteFile(configPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

    fmt.Printf("Configuration saved to %s\n", configPath)
    return nil
}

func getConfigPath() string {
    if path := os.Getenv("MYTOOL_CONFIG"); path != "" {
        return path
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".mytool", "config.yaml")
}
```

## Example Config File

```yaml
# ~/.mytool/config.yaml
api:
  base_url: https://api.production.com
  timeout: 60s
```

## Configuration Testing

```go
// internal/config/config_test.go
package config

import (
    "os"
    "testing"
)

func TestApplyEnvOverrides(t *testing.T) {
    tests := []struct {
        name     string
        envVars  map[string]string
        initial  Config
        expected Config
    }{
        {
            name: "override base URL",
            envVars: map[string]string{
                "MYTOOL_API_BASE_URL": "https://custom.com",
            },
            initial: Config{
                API: APIConfig{
                    BaseURL: "https://default.com",
                    Timeout: "30s",
                },
            },
            expected: Config{
                API: APIConfig{
                    BaseURL: "https://custom.com",
                    Timeout: "30s",
                },
            },
        },
        {
            name: "override timeout",
            envVars: map[string]string{
                "MYTOOL_API_TIMEOUT": "90s",
            },
            initial: Config{
                API: APIConfig{
                    BaseURL: "https://default.com",
                    Timeout: "30s",
                },
            },
            expected: Config{
                API: APIConfig{
                    BaseURL: "https://default.com",
                    Timeout: "90s",
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Set environment variables
            for k, v := range tt.envVars {
                os.Setenv(k, v)
                defer os.Unsetenv(k)
            }

            cfg := tt.initial
            applyEnvOverrides(&cfg)

            if cfg.API.BaseURL != tt.expected.API.BaseURL {
                t.Errorf("BaseURL = %v, want %v", cfg.API.BaseURL, tt.expected.API.BaseURL)
            }
            if cfg.API.Timeout != tt.expected.API.Timeout {
                t.Errorf("Timeout = %v, want %v", cfg.API.Timeout, tt.expected.API.Timeout)
            }
        })
    }
}
```

## Key Patterns

### Explicit Config Injection
```go
func Command(cfg *config.Config) (*cobra.Command, error)
```
No globals, config passed explicitly to all commands.

### Single Initialization Point
```go
func main() {
    config.Initialize()
    cfg := config.GetConfig()
    // ...
}
```
Config initialized once at startup, then passed around.

### Environment Variable Naming
```
MYTOOL_API_BASE_URL
MYTOOL_API_TIMEOUT
MYTOOL_LOG_LEVEL
```
Consistent prefix, uppercase, underscores.

### Validation
```go
func runCommand(cmd *cobra.Command, cfg *config.Config) error {
    if cfg.API.BaseURL == "" {
        return fmt.Errorf("API base URL not configured")
    }
    // ...
}
```
Validate config before using it in commands.

## Benefits

1. **No Hidden State**: Config passed explicitly, easy to test
2. **Clear Precedence**: Documented order of overrides
3. **Environment Friendly**: Standard environment variable pattern
4. **Testable**: Mock config easily in tests
5. **Type Safe**: Config is a struct, compiler checks
6. **User Friendly**: `config show` command shows current values

## Production Considerations

### Secrets Management
Don't store secrets in config file:

```go
// Get from environment or secret management system
apiKey := os.Getenv("MYTOOL_API_KEY")
if apiKey == "" {
    return fmt.Errorf("MYTOOL_API_KEY environment variable required")
}
```

### Config Validation
Validate config after loading:

```go
func (c *Config) Validate() error {
    if c.API.BaseURL == "" {
        return fmt.Errorf("api.base_url is required")
    }
    if _, err := time.ParseDuration(c.API.Timeout); err != nil {
        return fmt.Errorf("api.timeout must be valid duration: %w", err)
    }
    return nil
}
```

### Multiple Environments
Support different config profiles:

```bash
# Development
export MYTOOL_CONFIG=~/.mytool/config.dev.yaml

# Production
export MYTOOL_CONFIG=~/.mytool/config.prod.yaml
```
