package graph

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/db"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/twistingmercury/mnemonic/internal/metrics"
)

var tracer = otel.Tracer("mnemonic/neo4j")

// CypherRunner abstracts the ability to run Cypher queries.
// This interface is satisfied by neo4j.ManagedTransaction and can be
// implemented by test mocks without needing unexported methods.
type CypherRunner interface {
	Run(ctx context.Context, cypher string, params map[string]any) (ResultCollector, error)
}

// ResultCollector abstracts the ability to collect results from a Cypher query.
// This interface wraps the subset of neo4j.ResultWithContext that this package uses.
type ResultCollector interface {
	Collect(ctx context.Context) ([]*db.Record, error)
}

// SessionExecutor abstracts Neo4j session operations for testability.
type SessionExecutor interface {
	ExecuteRead(ctx context.Context, work func(runner CypherRunner) (any, error)) (any, error)
	Close(ctx context.Context) error
}

// SessionFactory creates SessionExecutor instances.
// This allows the repository to be tested without a real Neo4j driver.
type SessionFactory func(ctx context.Context) SessionExecutor

// Repository defines the read-only knowledge graph queries the MCP tools need.
type Repository interface {
	// GetPatternConcepts returns all concepts linked to a pattern via MENTIONED_IN relationships.
	GetPatternConcepts(ctx context.Context, patternID uuid.UUID) ([]Concept, error)

	// FindRelatedPatterns finds patterns related to the given pattern using pre-computed RELATED_TO edges.
	// Results include similarity scores and shared concept names.
	FindRelatedPatterns(ctx context.Context, patternID uuid.UUID, limit int) ([]RelatedPattern, error)
}

// neo4jRepository is a Neo4j implementation of Repository.
type neo4jRepository struct {
	driver    neo4j.DriverWithContext
	database  string
	factory   SessionFactory
	dbMetrics *metrics.Database
}

// NewRepository creates a new Neo4j-backed Repository. Each operation gets a
// client span and, when dbMetrics is non-nil, a latency and error recording.
func NewRepository(driver neo4j.DriverWithContext, database string, dbMetrics *metrics.Database) Repository {
	r := &neo4jRepository{
		driver:    driver,
		database:  database,
		dbMetrics: dbMetrics,
	}
	r.factory = r.defaultSessionFactory
	return r
}

// instrument starts a client span for one Neo4j operation. The returned func
// ends the span and records the operation's latency and outcome; defer it
// before the session close so close errors are counted too.
func (r *neo4jRepository) instrument(ctx context.Context, op string) (context.Context, func(error)) {
	start := time.Now()
	ctx, span := tracer.Start(ctx, "neo4j "+op,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", "neo4j"),
			attribute.String("db.namespace", r.database),
			attribute.String("db.operation.name", op),
		),
	)
	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
		if r.dbMetrics != nil {
			r.dbMetrics.Record(ctx, "neo4j", op, time.Since(start), err)
		}
	}
}

// neo4jSessionAdapter wraps a neo4j.SessionWithContext to satisfy SessionExecutor.
type neo4jSessionAdapter struct {
	session neo4j.SessionWithContext
}

// neo4jCypherRunnerAdapter wraps a neo4j.ManagedTransaction to satisfy CypherRunner.
type neo4jCypherRunnerAdapter struct {
	tx neo4j.ManagedTransaction
}

// neo4jResultAdapter wraps a neo4j.ResultWithContext to satisfy ResultCollector.
type neo4jResultAdapter struct {
	result neo4j.ResultWithContext
}

func (a *neo4jResultAdapter) Collect(ctx context.Context) ([]*db.Record, error) {
	return a.result.Collect(ctx)
}

func (a *neo4jCypherRunnerAdapter) Run(ctx context.Context, cypher string, params map[string]any) (ResultCollector, error) {
	result, err := a.tx.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	return &neo4jResultAdapter{result: result}, nil
}

func (a *neo4jSessionAdapter) ExecuteRead(ctx context.Context, work func(runner CypherRunner) (any, error)) (any, error) {
	return a.session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return work(&neo4jCypherRunnerAdapter{tx: tx})
	})
}

func (a *neo4jSessionAdapter) Close(ctx context.Context) error {
	return a.session.Close(ctx)
}

// defaultSessionFactory creates a real Neo4j session wrapped in an adapter.
func (r *neo4jRepository) defaultSessionFactory(ctx context.Context) SessionExecutor {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: r.database,
	})
	return &neo4jSessionAdapter{session: session}
}

// GetPatternConcepts returns all concepts linked to a pattern via MENTIONED_IN relationships.
func (r *neo4jRepository) GetPatternConcepts(ctx context.Context, patternID uuid.UUID) (_ []Concept, err error) {
	if patternID == uuid.Nil {
		return nil, errors.New("patternID must not be nil UUID")
	}

	ctx, done := r.instrument(ctx, "GetPatternConcepts")
	defer func() { done(err) }()

	session := r.factory(ctx)
	defer func() {
		if closeErr := session.Close(ctx); closeErr != nil && err == nil {
			err = fmt.Errorf("closing session: %w", closeErr)
		}
	}()

	result, err := session.ExecuteRead(ctx, func(runner CypherRunner) (any, error) {
		res, err := runner.Run(ctx,
			`MATCH (c:Concept)-[:MENTIONED_IN]->(p:Pattern {id: $patternId})
			 RETURN c.name AS name, c.type AS type
			 ORDER BY c.name`,
			map[string]any{"patternId": patternID.String()},
		)
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		if err != nil {
			return nil, err
		}

		concepts := make([]Concept, 0, len(records))
		for _, record := range records {
			nameVal, ok := record.Get("name")
			if !ok {
				return nil, fmt.Errorf("missing 'name' field in record")
			}
			nameStr, ok := nameVal.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected type for 'name': %T", nameVal)
			}

			typeVal, ok := record.Get("type")
			if !ok {
				return nil, fmt.Errorf("missing 'type' field in record")
			}
			typeStr, ok := typeVal.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected type for 'type': %T", typeVal)
			}

			concepts = append(concepts, Concept{
				Name: nameStr,
				Type: typeStr,
			})
		}

		return concepts, nil
	})

	if err != nil {
		return nil, fmt.Errorf("getting concepts for pattern %s: %w", patternID, err)
	}

	return result.([]Concept), nil
}

// FindRelatedPatterns finds patterns related to the given pattern using pre-computed RELATED_TO edges.
// Results include similarity scores and shared concept names.
func (r *neo4jRepository) FindRelatedPatterns(ctx context.Context, patternID uuid.UUID, limit int) (_ []RelatedPattern, err error) {
	if patternID == uuid.Nil {
		return nil, errors.New("patternID must not be nil UUID")
	}

	ctx, done := r.instrument(ctx, "FindRelatedPatterns")
	defer func() { done(err) }()

	session := r.factory(ctx)
	defer func() {
		if closeErr := session.Close(ctx); closeErr != nil && err == nil {
			err = fmt.Errorf("closing session: %w", closeErr)
		}
	}()

	result, err := session.ExecuteRead(ctx, func(runner CypherRunner) (any, error) {
		res, err := runner.Run(ctx,
			`MATCH (p1:Pattern {id: $patternId})-[r:RELATED_TO]-(p2:Pattern)
			 WITH p1, p2, r.similarity AS similarity
			 OPTIONAL MATCH (p1)<-[:MENTIONED_IN]-(c:Concept)-[:MENTIONED_IN]->(p2)
			 WITH p2, similarity, collect(c.name) AS conceptNames, count(c) AS sharedConcepts
			 ORDER BY similarity DESC
			 LIMIT $limit
			 RETURN p2.id AS id, p2.name AS name, sharedConcepts, similarity, conceptNames`,
			map[string]any{
				"patternId": patternID.String(),
				"limit":     limit,
			},
		)
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		if err != nil {
			return nil, err
		}

		patterns := make([]RelatedPattern, 0, len(records))
		for _, record := range records {
			idVal, ok := record.Get("id")
			if !ok {
				return nil, fmt.Errorf("missing 'id' field in record")
			}
			idStr, ok := idVal.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected type for 'id': %T", idVal)
			}

			nameVal, ok := record.Get("name")
			if !ok {
				return nil, fmt.Errorf("missing 'name' field in record")
			}
			nameStr, ok := nameVal.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected type for 'name': %T", nameVal)
			}

			sharedVal, ok := record.Get("sharedConcepts")
			if !ok {
				return nil, fmt.Errorf("missing 'sharedConcepts' field in record")
			}
			sharedInt, ok := sharedVal.(int64)
			if !ok {
				return nil, fmt.Errorf("unexpected type for 'sharedConcepts': %T", sharedVal)
			}

			similarityVal, ok := record.Get("similarity")
			if !ok {
				return nil, fmt.Errorf("missing 'similarity' field in record")
			}
			similarityFloat, ok := similarityVal.(float64)
			if !ok {
				return nil, fmt.Errorf("unexpected type for 'similarity': %T", similarityVal)
			}

			conceptNamesVal, ok := record.Get("conceptNames")
			if !ok {
				return nil, fmt.Errorf("missing 'conceptNames' field in record")
			}
			conceptNamesRaw, ok := conceptNamesVal.([]any)
			if !ok {
				return nil, fmt.Errorf("unexpected type for 'conceptNames': %T", conceptNamesVal)
			}
			conceptNames := make([]string, 0, len(conceptNamesRaw))
			for _, v := range conceptNamesRaw {
				s, ok := v.(string)
				if !ok {
					return nil, fmt.Errorf("unexpected type for concept name element: %T", v)
				}
				conceptNames = append(conceptNames, s)
			}

			parsedID, err := uuid.Parse(idStr)
			if err != nil {
				return nil, fmt.Errorf("parsing pattern ID %q: %w", idStr, err)
			}

			patterns = append(patterns, RelatedPattern{
				ID:             parsedID,
				Name:           nameStr,
				SharedConcepts: int(sharedInt),
				Similarity:     similarityFloat,
				ConceptNames:   conceptNames,
			})
		}

		return patterns, nil
	})

	if err != nil {
		return nil, fmt.Errorf("finding related patterns for %s: %w", patternID, err)
	}

	return result.([]RelatedPattern), nil
}
