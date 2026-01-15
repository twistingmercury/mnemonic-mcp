---
entity_name: Cobra Subcommand Pattern
entity_type: CLI Pattern
language: go
domain: cli
description: Subcommand implementation pattern with XCommand(cfg) function, separate run function with step comments, YAML file loading, validation, and API calls
tags:
  - Cobra
  - CLI
  - subcommands
  - validation
  - Go
---

# Cobra Subcommand Pattern

This pattern demonstrates implementing Cobra subcommands with explicit config injection, YAML file loading, validation, and step-by-step execution logic.

## Subcommand Structure

Every subcommand follows this structure:

1. **XCommand(cfg) function** - Returns configured cobra.Command
2. **Flag declarations** - Scoped to function
3. **RunE handler** - Calls separate run function
4. **runX() function** - Step-by-step implementation with comments

## Complete Subcommand Example

```go
// internal/cli/company/add.go
package company

import (
    "context"
    "fmt"
    "os"
    "time"
    "yourproject/internal/api"
    "yourproject/internal/config"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

// AddCommand returns the company add command
func AddCommand(cfg *config.Config) (*cobra.Command, error) {
    // Declare flag variables in function scope
    var yamlFile string

    cmd := &cobra.Command{
        Use:   "add",
        Short: "Add a company to the system",
        Long:  `Add a new company account to the system with specified attributes`,
        RunE: func(command *cobra.Command, args []string) error {
            // Delegate to separate run function for better testing
            return runCompanyAdd(command, cfg, yamlFile)
        },
    }

    // Define flags
    cmd.Flags().StringVar(&yamlFile, "yaml", "", "Path to company YAML file (required)")
    if err := cmd.MarkFlagRequired("yaml"); err != nil {
        return nil, fmt.Errorf("failed to mark yaml flag as required: %w", err)
    }

    return cmd, nil
}

func runCompanyAdd(command *cobra.Command, cfg *config.Config, yamlFile string) error {
    // Step 1: Check if --yaml flag is provided
    if yamlFile == "" {
        if err := command.Help(); err != nil {
            fmt.Printf("warning: failed to display help: %v\n", err)
        }
        return fmt.Errorf("--yaml flag is required")
    }

    // Step 2: Check if YAML file exists
    if _, err := os.Stat(yamlFile); os.IsNotExist(err) {
        if err := command.Help(); err != nil {
            fmt.Printf("warning: failed to display help: %v\n", err)
        }
        return fmt.Errorf("YAML file does not exist: %s", yamlFile)
    }

    // Step 3: Check if YAML file has content
    fileContent, err := os.ReadFile(yamlFile)
    if err != nil {
        if err := command.Help(); err != nil {
            fmt.Printf("warning: failed to display help: %v\n", err)
        }
        return fmt.Errorf("failed to read YAML file: %w", err)
    }

    if len(fileContent) == 0 {
        if err := command.Help(); err != nil {
            fmt.Printf("warning: failed to display help: %v\n", err)
        }
        return fmt.Errorf("YAML file has no content: %s", yamlFile)
    }

    // Step 4: Unmarshal YAML into domain struct
    var companySettings CompanySettings
    if err := yaml.Unmarshal(fileContent, &companySettings); err != nil {
        if err := command.Help(); err != nil {
            fmt.Printf("warning: failed to display help: %v\n", err)
        }
        return fmt.Errorf("failed to parse YAML file: %w", err)
    }

    // Step 5: Validate field values
    if err := ValidateCompanySettings(companySettings); err != nil {
        if err := command.Help(); err != nil {
            fmt.Printf("warning: failed to display help: %v\n", err)
        }
        return err
    }

    // Step 6: Check if API configuration is available
    if cfg.API.BaseURL == "" {
        return fmt.Errorf("API base URL not configured. Please run 'mytool config set --base-url <url>' first")
    }

    // Step 7: Create API client
    timeout, err := time.ParseDuration(cfg.API.Timeout)
    if err != nil {
        return fmt.Errorf("invalid timeout duration: %w", err)
    }

    // Check for --skip-auth persistent flag
    skipAuth, _ := command.Flags().GetBool("skip-auth")

    apiClient := api.NewClient(api.Config{
        BaseURL:  cfg.API.BaseURL,
        Timeout:  timeout,
        SkipAuth: skipAuth,
    })

    ctx := context.Background()

    // Step 8: Invoke the Company API to create the company
    companyReq := api.CompanyCreateRequest{
        CompanyGuid:  companySettings.CompanyGuid,
        Name:         companySettings.Name,
        EmailDomains: companySettings.EmailDomains,
    }

    _, err = apiClient.CreateCompany(ctx, companyReq)
    if err != nil {
        return fmt.Errorf("failed to create company: %w", err)
    }

    // Step 9: If additional data provided, make additional API calls
    if len(companySettings.Claims) > 0 {
        var claimsReq []api.CompanyClaimRequest
        for _, claim := range companySettings.Claims {
            claimsReq = append(claimsReq, api.CompanyClaimRequest{
                Claim: claim.Claim,
                Value: claim.Value,
            })
        }

        if err := apiClient.AddCompanyClaims(ctx, companySettings.CompanyGuid, claimsReq); err != nil {
            return fmt.Errorf("failed to add company claims: %w", err)
        }
    }

    // Step 10: Return success message
    fmt.Printf("Successfully created company: %s\n", companySettings.Name)
    if len(companySettings.Claims) > 0 {
        fmt.Printf("Added %d claim(s) to the company\n", len(companySettings.Claims))
    }

    return nil
}
```

## Find/Get Command Pattern

For retrieval commands:

```go
// internal/cli/company/find.go
package company

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    "yourproject/internal/api"
    "yourproject/internal/config"
    "github.com/spf13/cobra"
)

func FindCommand(cfg *config.Config) (*cobra.Command, error) {
    var companyGuid string
    var outputFormat string

    cmd := &cobra.Command{
        Use:   "find",
        Short: "Find a company by GUID",
        Long:  `Retrieve company details by company GUID`,
        RunE: func(command *cobra.Command, args []string) error {
            return runCompanyFind(command, cfg, companyGuid, outputFormat)
        },
    }

    cmd.Flags().StringVar(&companyGuid, "guid", "", "Company GUID (required)")
    if err := cmd.MarkFlagRequired("guid"); err != nil {
        return nil, fmt.Errorf("failed to mark guid flag as required: %w", err)
    }

    cmd.Flags().StringVar(&outputFormat, "output", "text", "Output format (text, json, yaml)")

    return cmd, nil
}

func runCompanyFind(command *cobra.Command, cfg *config.Config, companyGuid, outputFormat string) error {
    // Step 1: Validate GUID format
    if err := ValidateGuid(companyGuid); err != nil {
        return err
    }

    // Step 2: Create API client
    timeout, err := time.ParseDuration(cfg.API.Timeout)
    if err != nil {
        return fmt.Errorf("invalid timeout duration: %w", err)
    }

    skipAuth, _ := command.Flags().GetBool("skip-auth")
    apiClient := api.NewClient(api.Config{
        BaseURL:  cfg.API.BaseURL,
        Timeout:  timeout,
        SkipAuth: skipAuth,
    })

    ctx := context.Background()

    // Step 3: Fetch company from API
    company, err := apiClient.GetCompany(ctx, companyGuid)
    if err != nil {
        return fmt.Errorf("failed to get company: %w", err)
    }

    // Step 4: Format and display output
    switch outputFormat {
    case "json":
        data, err := json.MarshalIndent(company, "", "  ")
        if err != nil {
            return fmt.Errorf("failed to marshal JSON: %w", err)
        }
        fmt.Println(string(data))
    case "yaml":
        data, err := yaml.Marshal(company)
        if err != nil {
            return fmt.Errorf("failed to marshal YAML: %w", err)
        }
        fmt.Println(string(data))
    case "text":
        fmt.Printf("Company GUID:  %s\n", company.CompanyGuid)
        fmt.Printf("Name:          %s\n", company.Name)
        fmt.Printf("Email Domains: %v\n", company.EmailDomains)
    default:
        return fmt.Errorf("unknown output format: %s", outputFormat)
    }

    return nil
}
```

## Update Command Pattern

For update commands:

```go
// internal/cli/company/update.go
package company

func UpdateCommand(cfg *config.Config) (*cobra.Command, error) {
    var yamlFile string

    cmd := &cobra.Command{
        Use:   "update",
        Short: "Update an existing company",
        Long:  `Update company attributes using a YAML file`,
        RunE: func(command *cobra.Command, args []string) error {
            return runCompanyUpdate(command, cfg, yamlFile)
        },
    }

    cmd.Flags().StringVar(&yamlFile, "yaml", "", "Path to company update YAML file (required)")
    if err := cmd.MarkFlagRequired("yaml"); err != nil {
        return nil, fmt.Errorf("failed to mark yaml flag as required: %w", err)
    }

    return cmd, nil
}

func runCompanyUpdate(command *cobra.Command, cfg *config.Config, yamlFile string) error {
    // Step 1: Load and validate YAML (similar to add)
    fileContent, err := os.ReadFile(yamlFile)
    if err != nil {
        return fmt.Errorf("failed to read YAML file: %w", err)
    }

    var updateSettings CompanyUpdateSettings
    if err := yaml.Unmarshal(fileContent, &updateSettings); err != nil {
        return fmt.Errorf("failed to parse YAML file: %w", err)
    }

    // Step 2: Validate update settings
    if err := ValidateCompanyUpdateSettings(updateSettings); err != nil {
        return err
    }

    // Step 3: Create API client
    timeout, err := time.ParseDuration(cfg.API.Timeout)
    if err != nil {
        return fmt.Errorf("invalid timeout duration: %w", err)
    }

    skipAuth, _ := command.Flags().GetBool("skip-auth")
    apiClient := api.NewClient(api.Config{
        BaseURL:  cfg.API.BaseURL,
        Timeout:  timeout,
        SkipAuth: skipAuth,
    })

    ctx := context.Background()

    // Step 4: Call update API
    updateReq := api.CompanyUpdateRequest{
        Name:         updateSettings.Name,
        EmailDomains: updateSettings.EmailDomains,
    }

    if err := apiClient.UpdateCompany(ctx, updateSettings.CompanyGuid, updateReq); err != nil {
        return fmt.Errorf("failed to update company: %w", err)
    }

    // Step 5: Success message
    fmt.Printf("Successfully updated company: %s\n", updateSettings.CompanyGuid)

    return nil
}
```

## List Command Pattern

For list commands with pagination:

```go
// internal/cli/user/list.go
package user

func ListCommand(cfg *config.Config) (*cobra.Command, error) {
    var status string
    var page int
    var pageSize int
    var outputFormat string

    cmd := &cobra.Command{
        Use:   "list",
        Short: "List users",
        Long:  `List users with optional filtering and pagination`,
        RunE: func(command *cobra.Command, args []string) error {
            return runUserList(command, cfg, status, page, pageSize, outputFormat)
        },
    }

    cmd.Flags().StringVar(&status, "status", "", "Filter by status (active, inactive)")
    cmd.Flags().IntVar(&page, "page", 1, "Page number")
    cmd.Flags().IntVar(&pageSize, "page-size", 20, "Items per page")
    cmd.Flags().StringVar(&outputFormat, "output", "text", "Output format (text, json)")

    return cmd, nil
}

func runUserList(command *cobra.Command, cfg *config.Config, status string, page, pageSize int, outputFormat string) error {
    // Step 1: Validate pagination parameters
    if page < 1 {
        return fmt.Errorf("page must be >= 1")
    }
    if pageSize < 1 || pageSize > 100 {
        return fmt.Errorf("page-size must be between 1 and 100")
    }

    // Step 2: Create API client
    timeout, err := time.ParseDuration(cfg.API.Timeout)
    if err != nil {
        return fmt.Errorf("invalid timeout duration: %w", err)
    }

    skipAuth, _ := command.Flags().GetBool("skip-auth")
    apiClient := api.NewClient(api.Config{
        BaseURL:  cfg.API.BaseURL,
        Timeout:  timeout,
        SkipAuth: skipAuth,
    })

    ctx := context.Background()

    // Step 3: Build query parameters
    params := api.ListUsersParams{
        Page:     page,
        PageSize: pageSize,
        Status:   status,
    }

    // Step 4: Fetch users
    users, pagination, err := apiClient.ListUsers(ctx, params)
    if err != nil {
        return fmt.Errorf("failed to list users: %w", err)
    }

    // Step 5: Format output
    if outputFormat == "json" {
        data, err := json.MarshalIndent(users, "", "  ")
        if err != nil {
            return fmt.Errorf("failed to marshal JSON: %w", err)
        }
        fmt.Println(string(data))
    } else {
        // Text table format
        fmt.Printf("%-40s %-30s %-15s\n", "USER GUID", "EMAIL", "STATUS")
        fmt.Println(strings.Repeat("-", 90))
        for _, user := range users {
            fmt.Printf("%-40s %-30s %-15s\n", user.UserGuid, user.Email, user.Status)
        }
        fmt.Println()
        fmt.Printf("Page %d of %d (Total: %d users)\n", pagination.Page, pagination.TotalPages, pagination.TotalItems)
    }

    return nil
}
```

## Key Patterns

### XCommand(cfg) Function Signature
```go
func AddCommand(cfg *config.Config) (*cobra.Command, error)
```
Returns configured command with error for initialization failures.

### Flag Variables in Function Scope
```go
func AddCommand(cfg *config.Config) (*cobra.Command, error) {
    var yamlFile string  // Scoped to function
    // ...
}
```
Not package-level globals, cleaner testing.

### Separate Run Function
```go
RunE: func(command *cobra.Command, args []string) error {
    return runCompanyAdd(command, cfg, yamlFile)
}
```
Separates command setup from execution logic.

### Step-by-Step Comments
```go
// Step 1: Check if --yaml flag is provided
// Step 2: Check if YAML file exists
// Step 3: Check if YAML file has content
```
Makes code self-documenting and easy to follow.

### Help on Error
```go
if yamlFile == "" {
    if err := command.Help(); err != nil {
        fmt.Printf("warning: failed to display help: %v\n", err)
    }
    return fmt.Errorf("--yaml flag is required")
}
```
Show help when user makes usage error.

### Persistent Flag Access
```go
skipAuth, _ := command.Flags().GetBool("skip-auth")
```
Access root-level persistent flags in subcommands.

## Testing Subcommands

```go
// internal/cli/company/add_test.go
package company

import (
    "testing"
    "yourproject/internal/config"
)

func TestAddCommand(t *testing.T) {
    cfg := &config.Config{
        API: config.APIConfig{
            BaseURL: "https://test.example.com",
            Timeout: "30s",
        },
    }

    cmd, err := AddCommand(cfg)
    if err != nil {
        t.Fatalf("AddCommand() error = %v", err)
    }

    if cmd.Use != "add" {
        t.Errorf("Use = %v, want %v", cmd.Use, "add")
    }

    // Test flag is defined
    flag := cmd.Flags().Lookup("yaml")
    if flag == nil {
        t.Error("yaml flag not defined")
    }
}

func TestRunCompanyAdd_YamlFileRequired(t *testing.T) {
    cfg := &config.Config{}
    cmd, _ := AddCommand(cfg)

    err := runCompanyAdd(cmd, cfg, "")
    if err == nil {
        t.Error("expected error for missing yaml file")
    }
}
```

## Benefits

1. **Consistent Structure**: All subcommands follow same pattern
2. **Testable**: Separate functions easy to unit test
3. **Clear Flow**: Step comments make execution clear
4. **Config Injection**: No hidden dependencies
5. **Validation Early**: Check inputs before API calls
6. **User Friendly**: Help shown on usage errors
7. **Error Handling**: Detailed error messages with context
