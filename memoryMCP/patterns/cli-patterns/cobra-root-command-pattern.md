---
entity_name: Cobra Root Command Pattern
entity_type: CLI Pattern
language: go
domain: cli
description: Root command pattern with custom exit codes, error mapping, custom help template with environment variables, and explicit command initialization
tags:
  - Cobra
  - CLI
  - root-command
  - error-handling
  - Go
---

# Cobra Root Command Pattern

This pattern demonstrates setting up a Cobra root command with custom exit codes, error-to-exit-code mapping, custom help templates, and explicit initialization.

## Root Command Setup

```go
// internal/cli/root/root.go
package root

import (
    "errors"
    "fmt"
    "os"
    "strings"
    "github.com/spf13/cobra"
)

// Custom exit codes following sysexits.h conventions
const (
    ExOk         = 0   // Successful execution
    ExitError    = 1   // General error
    ExitUsage    = 2   // Command line usage error
    ExitNoInput  = 66  // Input file/data not found
    ExitTempFail = 75  // Temporary failure (timeout, etc.)
    ExitNoPerm   = 77  // Permission denied
)

var rootCmd = &cobra.Command{
    Use:           `mytool`,
    Short:         "CLI tool for system management",
    Long:          `A command line interface for managing resources in the system.`,
    Version:       version.Version(),
    SilenceUsage:  true,  // Don't show usage on errors
    SilenceErrors: true,  // Handle errors ourselves
}

// Initialize bootstraps the root command with all of the sub commands
func Initialize(subCommands ...*cobra.Command) error {
    for _, cmd := range subCommands {
        rootCmd.AddCommand(cmd)
    }

    // Add global persistent flags
    rootCmd.PersistentFlags().Bool("skip-auth", false, "Skip the authorization check")

    rootCmd.SetOut(os.Stdout)
    rootCmd.SetErr(os.Stderr)
    rootCmd.SetHelpTemplate(customHelpTemplate)

    return nil
}

// Execute is the true entrypoint to run the commands
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        _, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(DetermineExitCode(err))
    }
}
```

## Custom Help Template with Environment Variables

```go
// customHelpTemplate adds environment variables section to the default Cobra help template
const customHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

ENVIRONMENT VARIABLES:
  MYTOOL_API_BASE_URL     Override the API base URL from configuration
  MYTOOL_API_TIMEOUT      Override the request timeout (e.g., '30s', '5m')
  MYTOOL_LOG_LEVEL        Set log level (debug, info, warn, error)

Configuration precedence: config file < environment variables < command flags
{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
```

## Error to Exit Code Mapping

```go
// DetermineExitCode maps errors to appropriate exit codes
func DetermineExitCode(err error) int {
    if err == nil {
        return ExOk
    }

    // Handle specific error types
    switch {
    case errors.Is(err, os.ErrNotExist):
        return ExitNoInput // 66 - File not found
    case errors.Is(err, os.ErrPermission):
        return ExitNoPerm // 77 - Permission denied
    case errors.Is(err, os.ErrDeadlineExceeded):
        return ExitTempFail // 75 - Timeout/temporary failure
    }

    // Handle validation/usage errors by string matching
    if strings.Contains(err.Error(), "required") ||
        strings.Contains(err.Error(), "must be") ||
        strings.Contains(err.Error(), "invalid") {
        return ExitUsage // 2 - Invalid usage
    }

    return ExitError // 1 - General error
}
```

## Exit Code Testing

```go
// internal/cli/root/root_test.go
package root

import (
    "errors"
    "os"
    "testing"
)

func TestDetermineExitCode(t *testing.T) {
    tests := []struct {
        name string
        err  error
        want int
    }{
        {
            name: "nil error returns ExOk",
            err:  nil,
            want: ExOk,
        },
        {
            name: "file not found returns ExitNoInput",
            err:  os.ErrNotExist,
            want: ExitNoInput,
        },
        {
            name: "permission denied returns ExitNoPerm",
            err:  os.ErrPermission,
            want: ExitNoPerm,
        },
        {
            name: "timeout returns ExitTempFail",
            err:  os.ErrDeadlineExceeded,
            want: ExitTempFail,
        },
        {
            name: "required field error returns ExitUsage",
            err:  errors.New("field is required"),
            want: ExitUsage,
        },
        {
            name: "invalid field error returns ExitUsage",
            err:  errors.New("invalid value"),
            want: ExitUsage,
        },
        {
            name: "must be error returns ExitUsage",
            err:  errors.New("value must be valid"),
            want: ExitUsage,
        },
        {
            name: "generic error returns ExitError",
            err:  errors.New("something went wrong"),
            want: ExitError,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := DetermineExitCode(tt.err); got != tt.want {
                t.Errorf("DetermineExitCode() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Persistent Flags Pattern

Persistent flags are available to the command and all its subcommands:

```go
func Initialize(subCommands ...*cobra.Command) error {
    // ... add subcommands ...

    // Global flags available to all commands
    rootCmd.PersistentFlags().Bool("skip-auth", false, "Skip the authorization check")
    rootCmd.PersistentFlags().String("config", "", "Config file path (default ~/.mytool/config.yaml)")
    rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
    rootCmd.PersistentFlags().String("output", "text", "Output format (text, json, yaml)")

    return nil
}
```

Accessing persistent flags in subcommands:

```go
func runCommand(cmd *cobra.Command, args []string) error {
    // Access persistent flag
    skipAuth, _ := cmd.Flags().GetBool("skip-auth")
    debug, _ := cmd.Flags().GetBool("debug")

    if debug {
        log.SetLevel(log.DebugLevel)
    }

    // Use flags...
    return nil
}
```

## Version Command

Cobra can automatically add version information:

```go
// cmd/version/version.go
package version

var (
    Version   string = "dev"
    GitCommit string = "unknown"
    BuildDate string = "unknown"
)

func Version() string {
    return fmt.Sprintf("%s (commit: %s, built: %s)", Version, GitCommit, BuildDate)
}
```

Set version in root command:

```go
var rootCmd = &cobra.Command{
    Use:     `mytool`,
    Version: version.Version(),
    // ...
}
```

Build with version information:

```bash
go build -ldflags "\
  -X 'main/cmd/version.Version=1.2.3' \
  -X 'main/cmd/version.GitCommit=$(git rev-parse --short HEAD)' \
  -X 'main/cmd/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
  -o mytool ./cmd/main
```

## Key Patterns

### SilenceUsage and SilenceErrors
```go
SilenceUsage:  true,  // Don't auto-show usage on errors
SilenceErrors: true,  // Handle errors ourselves in Execute()
```
This gives you full control over error formatting and when to show usage.

### Custom Exit Codes
Follow sysexits.h conventions for better shell script integration:
- 0: Success
- 1: General error
- 2: Usage error (wrong flags, missing required args)
- 66: Input not found (file doesn't exist)
- 75: Temporary failure (timeout, retry possible)
- 77: Permission denied

### Error Handling Philosophy
```go
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        // Print error to stderr
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        // Exit with appropriate code
        os.Exit(DetermineExitCode(err))
    }
}
```

### Environment Variables in Help
Document environment variables in custom help template so users know about configuration options.

## Shell Integration

Exit codes enable proper shell error handling:

```bash
#!/bin/bash
set -e  # Exit on error

mytool user add --yaml user.yaml
if [ $? -eq 2 ]; then
    echo "Usage error - check your flags"
    exit 1
elif [ $? -eq 66 ]; then
    echo "File not found"
    exit 1
fi

# Success
echo "User created successfully"
```

## Testing Strategy

Test exit code mapping thoroughly:

```go
func TestExecuteWithError(t *testing.T) {
    // Test that errors result in proper exit codes
    // Mock commands, trigger errors, verify exit codes
}
```

## Production Considerations

### Logging
Add structured logging at root level:

```go
func Initialize(subCommands ...*cobra.Command) error {
    // Setup logging based on debug flag
    rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
        debug, _ := cmd.Flags().GetBool("debug")
        if debug {
            log.SetLevel(log.DebugLevel)
        }
    }
    // ...
}
```

### Context
Pass context through commands for cancellation:

```go
func Execute() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        cancel()
    }()

    if err := rootCmd.ExecuteContext(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(DetermineExitCode(err))
    }
}
```

### Telemetry
Add telemetry hooks:

```go
func Initialize(subCommands ...*cobra.Command) error {
    rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
        // Send telemetry after command completes
        telemetry.RecordCommandExecution(cmd.Use)
    }
    // ...
}
```

## Benefits

1. **Explicit initialization**: Clear command wiring in main.go
2. **Custom exit codes**: Better shell script integration
3. **Error mapping**: Automatic exit code selection
4. **Custom help**: Document environment variables
5. **Testable**: All logic can be unit tested
6. **No magic**: No init() functions, everything explicit
