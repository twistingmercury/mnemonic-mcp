---
entity_name: Cobra Domain Architecture Pattern
entity_type: CLI Pattern
language: go
domain: cli
description: Domain-based Cobra CLI architecture organizing commands by business domain with dedicated directories, types, and validation per domain
tags:
  - Cobra
  - CLI
  - architecture
  - domain-driven
  - Go
---

# Cobra Domain Architecture Pattern

This pattern demonstrates organizing Cobra CLI commands by business domain rather than flat command structure, improving scalability and maintainability for complex CLIs.

## Directory Structure

```
project/
├── cmd/
│   └── main/
│       └── main.go              # Wiring domains to root
├── internal/
│   ├── cli/
│   │   ├── root/
│   │   │   ├── root.go          # Root command
│   │   │   └── root_test.go
│   │   ├── company/             # Company domain
│   │   │   ├── company.go       # Parent command + wiring
│   │   │   ├── add.go           # Subcommand
│   │   │   ├── add_test.go
│   │   │   ├── update.go        # Subcommand
│   │   │   ├── update_test.go
│   │   │   ├── find.go          # Subcommand
│   │   │   ├── find_test.go
│   │   │   ├── disable.go       # Subcommand
│   │   │   ├── disable_test.go
│   │   │   ├── claims.go        # Subcommand
│   │   │   ├── claims_test.go
│   │   │   ├── types.go         # Domain-specific types
│   │   │   └── validation.go    # Domain validation logic
│   │   ├── user/                # User domain
│   │   │   ├── user.go
│   │   │   ├── add.go
│   │   │   ├── list.go
│   │   │   ├── find.go
│   │   │   ├── update.go
│   │   │   ├── disable.go
│   │   │   ├── claims.go
│   │   │   ├── types.go
│   │   │   └── validation.go
│   │   └── config/              # Config domain
│   │       ├── config.go
│   │       └── config_test.go
│   ├── api/                     # API client
│   │   └── client.go
│   └── config/                  # Configuration management
│       └── config.go
```

## Parent Command Pattern

Each domain has a parent command that wires subcommands together:

```go
// internal/cli/company/company.go
package company

import (
    "fmt"
    "yourproject/internal/config"
    "github.com/spf13/cobra"
)

// Command returns the company command with all subcommands
func Command(cfg *config.Config) (*cobra.Command, error) {
    cmd := &cobra.Command{
        Use:   "company",
        Short: "Manage companies in the system",
        Long:  `Commands for managing company accounts in the identity system`,
    }

    // Add subcommands
    addCmd, err := AddCommand(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create add command: %w", err)
    }
    cmd.AddCommand(addCmd)

    updateCmd, err := UpdateCommand(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create update command: %w", err)
    }
    cmd.AddCommand(updateCmd)

    findCmd, err := FindCommand(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create find command: %w", err)
    }
    cmd.AddCommand(findCmd)

    disableCmd, err := DisableCommand(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create disable command: %w", err)
    }
    cmd.AddCommand(disableCmd)

    claimsCmd, err := ClaimsCommand(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create claims command: %w", err)
    }
    cmd.AddCommand(claimsCmd)

    return cmd, nil
}
```

## Domain Types Pattern

Each domain defines its own types for YAML input and API requests:

```go
// internal/cli/company/types.go
package company

// CompanySettings represents the YAML structure for company add operations
type CompanySettings struct {
    CompanyGuid  string         `yaml:"companyGuid"`
    Name         string         `yaml:"name"`
    EmailDomains []string       `yaml:"emailDomains"`
    Claims       []CompanyClaim `yaml:"claims"`
}

// CompanyUpdateSettings represents the YAML structure for company update operations
type CompanyUpdateSettings struct {
    CompanyGuid  string         `yaml:"companyGuid"`
    Name         string         `yaml:"name"`
    EmailDomains []string       `yaml:"emailDomains"`
    Claims       []CompanyClaim `yaml:"claims"`
}

// CompanyClaim represents a company claim with API-compatible field names
type CompanyClaim struct {
    Claim string `yaml:"Claim"` // Note: Capital 'C' to match API
    Value string `yaml:"Value"` // Note: Capital 'V' to match API
}
```

## Domain Validation Pattern

Each domain has dedicated validation logic with regex patterns:

```go
// internal/cli/company/validation.go
package company

import (
    "fmt"
    "regexp"
    "strings"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var domainRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// ValidateCompanySettings validates the CompanySettings struct according to requirements
func ValidateCompanySettings(settings CompanySettings) error {
    var errors []string

    // Validate companyGuid (must be valid UUID)
    if settings.CompanyGuid == "" {
        errors = append(errors, "companyGuid is required")
    } else if !uuidRegex.MatchString(strings.ToLower(settings.CompanyGuid)) {
        errors = append(errors, "companyGuid must be a valid UUID")
    }

    // Validate name (must be non-empty string, min 1, max 255 characters)
    if settings.Name == "" {
        errors = append(errors, "name is required")
    } else if len(settings.Name) > 255 {
        errors = append(errors, "name must be 255 characters or less")
    }

    // Validate emailDomains (must have at least one valid email domain)
    if len(settings.EmailDomains) == 0 {
        errors = append(errors, "emailDomains must have at least one domain")
    } else {
        for i, domain := range settings.EmailDomains {
            if domain == "" {
                errors = append(errors, fmt.Sprintf("email domain %d is empty", i+1))
            } else if !domainRegex.MatchString(domain) {
                errors = append(errors, fmt.Sprintf("email domain %d (%s) is not a valid domain format", i+1, domain))
            }
        }
    }

    // Validate claims (optional, but if provided each must have non-empty Claim and Value)
    for i, claim := range settings.Claims {
        if claim.Claim == "" {
            errors = append(errors, fmt.Sprintf("claim %d: Claim is required", i+1))
        } else if len(claim.Claim) > 100 {
            errors = append(errors, fmt.Sprintf("claim %d: Claim must be 100 characters or less", i+1))
        }

        if claim.Value == "" {
            errors = append(errors, fmt.Sprintf("claim %d: Value is required", i+1))
        } else if len(claim.Value) > 500 {
            errors = append(errors, fmt.Sprintf("claim %d: Value must be 500 characters or less", i+1))
        }
    }

    // Return aggregated errors
    if len(errors) > 0 {
        return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
    }

    return nil
}
```

## Main.go Wiring Pattern

Wire domains to root command in main.go:

```go
// cmd/main/main.go
package main

import (
    "fmt"
    "log"
    "yourproject/internal/cli/company"
    configCmd "yourproject/internal/cli/config"
    "yourproject/internal/cli/root"
    "yourproject/internal/cli/user"
    "yourproject/internal/config"
    "github.com/spf13/cobra"
)

func main() {
    if err := config.Initialize(); err != nil {
        log.Fatalln(fmt.Errorf("failed to initialize config: %w", err))
    }

    cfg := config.GetConfig()
    subcommands := make([]*cobra.Command, 0)

    // Add config command
    configCommand, err := configCmd.Command(&cfg)
    if err != nil {
        log.Fatalln(fmt.Errorf("failed to create config command: %w", err))
    }
    subcommands = append(subcommands, configCommand)

    // Add user command
    userCommand, err := user.Command(&cfg)
    if err != nil {
        log.Fatalln(fmt.Errorf("failed to create user command: %w", err))
    }
    subcommands = append(subcommands, userCommand)

    // Add company command
    companyCommand, err := company.Command(&cfg)
    if err != nil {
        log.Fatalln(fmt.Errorf("failed to create company command: %w", err))
    }
    subcommands = append(subcommands, companyCommand)

    if err := root.Initialize(subcommands...); err != nil {
        log.Fatalln(fmt.Errorf("failed to initialize root command: %w", err))
    }
    root.Execute()
}
```

## Key Patterns

### Domain Boundaries
- Each domain is a self-contained package
- Domain owns its types, validation, and commands
- Mirrors backend service boundaries (DDD)

### Explicit Dependencies
- Config passed explicitly to all commands
- No hidden globals or package-level state
- Easy to mock for testing

### Consistent Structure
- Every domain has same file organization
- types.go for domain models
- validation.go for validation logic
- Parent command wires subcommands

### Scalability
- Adding new subcommand: add file to domain directory
- Adding new domain: create new directory
- No growing "god file" with all commands

### Testability
- Each XCommand(cfg) can be unit tested
- Validation functions are pure (table-driven tests)
- Run functions can be tested with mock config

## Command Usage Examples

```bash
# Parent command shows subcommands
$ mytool company
Commands for managing company accounts in the identity system

Available Commands:
  add       Add a company to the system
  update    Update an existing company
  find      Find a company by GUID
  disable   Disable a company account
  claims    Manage company claims

# Subcommand with flags
$ mytool company add --yaml company.yaml

# Different domain
$ mytool user add --yaml user.yaml
$ mytool user list --status active
```

## Benefits

### For Development
- Clear code organization by business domain
- Easy to find: "company add" → company/add.go
- New developers understand structure immediately
- Reduced merge conflicts (different files)

### For Maintenance
- Changes isolated to specific domain
- Validation logic grouped with commands
- Types defined near usage

### For Testing
- Unit test domain functions independently
- Table-driven validation tests
- Mock config injection

### For Growth
- Scales to dozens of domains
- Scales to hundreds of subcommands
- Pattern is mechanical to follow
- No architectural refactoring needed

## When to Use

**Perfect for:**
- CRUD-heavy CLIs (identity, infrastructure, etc.)
- CLIs that mirror microservice architectures
- Multi-team CLIs (teams own domains)
- Long-lived CLIs that will grow significantly

**Not needed for:**
- Very simple CLIs (< 10 commands total)
- Single-domain tools
- Quick prototypes or scripts

## Comparison to Flat Structure

**Flat Cobra Structure:**
```
cmd/
├── root.go
├── company_add.go
├── company_update.go
├── company_find.go
├── user_add.go
├── user_update.go
└── user_list.go
```
Problems: Hard to navigate, no clear boundaries, grows unwieldy

**Domain-Based Structure:**
```
internal/cli/
├── company/
│   ├── add.go
│   ├── update.go
│   ├── find.go
│   ├── types.go
│   └── validation.go
└── user/
    ├── add.go
    ├── update.go
    ├── list.go
    ├── types.go
    └── validation.go
```
Benefits: Clear boundaries, scales well, easy navigation

## Testing Strategy

```go
// internal/cli/company/validation_test.go
package company

import "testing"

func TestValidateCompanySettings(t *testing.T) {
    tests := []struct {
        name        string
        settings    CompanySettings
        wantErr     bool
        errContains string
    }{
        {
            name: "valid company",
            settings: CompanySettings{
                CompanyGuid:  "123e4567-e89b-12d3-a456-426614174000",
                Name:         "Acme Corp",
                EmailDomains: []string{"acme.com"},
            },
            wantErr: false,
        },
        {
            name: "missing guid",
            settings: CompanySettings{
                Name:         "Acme Corp",
                EmailDomains: []string{"acme.com"},
            },
            wantErr:     true,
            errContains: "companyGuid is required",
        },
        {
            name: "invalid uuid",
            settings: CompanySettings{
                CompanyGuid:  "not-a-uuid",
                Name:         "Acme Corp",
                EmailDomains: []string{"acme.com"},
            },
            wantErr:     true,
            errContains: "must be a valid UUID",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateCompanySettings(tt.settings)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateCompanySettings() error = %v, wantErr %v", err, tt.wantErr)
            }
            if err != nil && tt.errContains != "" {
                if !strings.Contains(err.Error(), tt.errContains) {
                    t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
                }
            }
        })
    }
}
```

## Production Example

The midsctl CLI uses this pattern for managing identity services:
- Domains: company, user, config
- ~15 subcommands across 3 domains
- Clear separation of concerns
- Easy for team to contribute
