package routingrule

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

// MatchType is a string enum representing the type of match a routing rule uses.
// This is the authoritative definition used by both the repository and routing packages.
type MatchType string

// Valid MatchType values.
const (
	MatchTypeKeyword MatchType = "keyword"
	MatchTypeRegex   MatchType = "regex"
	MatchTypePattern MatchType = "pattern"
	MatchTypeDefault MatchType = "default"
)

// Rule represents a routing rule definition stored in the database.
type Rule struct {
	// ID is the unique identifier for the routing rule.
	ID uuid.UUID `db:"id"`

	// Name is a unique human-readable identifier for the rule.
	Name string `db:"name"`

	// Priority defines the evaluation order (0-1000, higher evaluated first).
	Priority int `db:"priority"`

	// AgentName is the target agent for this rule.
	AgentName string `db:"agent_name"`

	// MatchType determines how MatchConfig is interpreted.
	// Valid values: keyword, regex, pattern, default
	MatchType string `db:"match_type"`

	// MatchConfig contains type-specific match configuration.
	// The concrete type depends on MatchType.
	MatchConfig MatchConfig `db:"-"`

	// Enabled indicates whether this rule is active in routing evaluation.
	Enabled bool `db:"enabled"`

	// CreatedAt is the timestamp when the rule was created.
	CreatedAt time.Time `db:"created_at"`

	// UpdatedAt is the timestamp when the rule was last modified.
	UpdatedAt time.Time `db:"updated_at"`
}

// ValidMatchTypes defines the valid values for the MatchType field.
// These are derived from the MatchType constants to avoid duplication.
var ValidMatchTypes = []string{
	string(MatchTypeKeyword),
	string(MatchTypeRegex),
	string(MatchTypePattern),
	string(MatchTypeDefault),
}

// IsValidMatchType checks if the given match type string is valid.
func IsValidMatchType(matchType string) bool {
	return slices.Contains(ValidMatchTypes, matchType)
}

// MatchMode is a string enum representing the keyword match mode for a routing rule.
type MatchMode string

// Valid MatchMode values.
const (
	MatchModeAny MatchMode = "any"
	MatchModeAll MatchMode = "all"
)

// ValidMatchModes defines the valid values for keyword match mode.
var ValidMatchModes = []MatchMode{MatchModeAny, MatchModeAll}

// IsValidMatchMode checks if the given match mode is valid.
func IsValidMatchMode(mode MatchMode) bool {
	return slices.Contains(ValidMatchModes, mode)
}

// MatchConfig is the interface for type-specific match configurations.
// Use type assertion to access type-specific fields.
type MatchConfig interface {
	// Type returns the match type identifier.
	Type() string
}

// KeywordMatchConfig is the configuration for match_type = 'keyword'.
type KeywordMatchConfig struct {
	// Keywords is the list of keywords to match against.
	Keywords []string `json:"keywords"`

	// MatchMode determines how keywords are matched: "any" or "all".
	MatchMode MatchMode `json:"match_mode"`
}

// Type returns the match type identifier.
func (k KeywordMatchConfig) Type() string { return "keyword" }

// RegexMatchConfig is the configuration for match_type = 'regex'.
type RegexMatchConfig struct {
	// Pattern is the regular expression pattern to match against.
	Pattern string `json:"pattern"`

	// Flags contains regex flags (e.g., "i" for case-insensitive).
	Flags string `json:"flags,omitempty"`
}

// Type returns the match type identifier.
func (r RegexMatchConfig) Type() string { return "regex" }

// PatternMatchConfig is the configuration for match_type = 'pattern'.
// NOTE: The service layer is responsible for validating that all pattern IDs
// in PatternIDs actually exist in the patterns table. The repository layer
// does not enforce this constraint as routing_rules.match_config is stored
// as JSONB without foreign key relationships.
type PatternMatchConfig struct {
	// PatternIDs is the list of pattern UUIDs for semantic matching.
	PatternIDs []uuid.UUID `json:"pattern_ids"`
}

// Type returns the match type identifier.
func (p PatternMatchConfig) Type() string { return "pattern" }

// DefaultMatchConfig is the configuration for match_type = 'default'.
// The default match type always matches and is used as a fallback.
type DefaultMatchConfig struct{}

// Type returns the match type identifier.
func (d DefaultMatchConfig) Type() string { return "default" }

// UnmarshalMatchConfig unmarshals a JSONB match_config based on the match_type.
func UnmarshalMatchConfig(matchType string, data []byte) (MatchConfig, error) {
	switch matchType {
	case "keyword":
		var cfg KeywordMatchConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshaling keyword config: %w", err)
		}
		return cfg, nil

	case "regex":
		var cfg RegexMatchConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshaling regex config: %w", err)
		}
		return cfg, nil

	case "pattern":
		var cfg PatternMatchConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshaling pattern config: %w", err)
		}
		return cfg, nil

	case "default":
		return DefaultMatchConfig{}, nil

	default:
		return nil, fmt.Errorf("unknown match type: %s", matchType)
	}
}

// MarshalMatchConfig marshals a MatchConfig to JSON bytes.
func MarshalMatchConfig(cfg MatchConfig) ([]byte, error) {
	if cfg == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(cfg)
}

// Filter defines filtering options for rule queries.
type Filter struct {
	// AgentName filters by target agent name.
	AgentName *string

	// MatchType filters by match type.
	MatchType *string

	// Enabled filters by enabled state.
	Enabled *bool
}
