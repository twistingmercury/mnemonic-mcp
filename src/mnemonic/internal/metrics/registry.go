// Package metrics provides centralized metric instrumentation for the Mnemonic service.
// It defines domain-specific metrics for routing, patterns, and database operations.
package metrics

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

// Registry holds all metric instruments for the application.
// It provides centralized access to domain-specific metrics.
type Registry struct {
	Routing  *Routing
	Patterns *Pattern
	Database *Database
}

// NewRegistry creates all metric instruments using the provided meter.
// It initializes routing, pattern, and database metrics.
func NewRegistry(meter metric.Meter) (*Registry, error) {
	routing, err := NewRouting(meter)
	if err != nil {
		return nil, fmt.Errorf("routing metrics: %w", err)
	}

	patterns, err := NewPattern(meter)
	if err != nil {
		return nil, fmt.Errorf("pattern metrics: %w", err)
	}

	database, err := NewDatabase(meter)
	if err != nil {
		return nil, fmt.Errorf("database metrics: %w", err)
	}

	return &Registry{
		Routing:  routing,
		Patterns: patterns,
		Database: database,
	}, nil
}
