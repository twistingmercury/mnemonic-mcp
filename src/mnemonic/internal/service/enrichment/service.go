// Package enrichment provides the business logic layer for processing enrichment
// jobs. It orchestrates the enrichment pipeline: claiming jobs, running embedding
// generation, concept extraction, Neo4j graph sync, and handling failures with
// retry scheduling.
//
// The enrichment worker goroutine calls these methods; REST/MCP handlers do not.
//
// Documentation:
//   - Design: docs/design/service-layer.md (EnrichmentService, Enrichment Job Lifecycle)
//   - Design: docs/design/pattern-processing.md (Enrichment Pipeline)
package enrichment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/twistingmercury/mnemonic/internal/config"
	agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
	enrichmentjob "github.com/twistingmercury/mnemonic/internal/repository/enrichmentjob"
	graphrepo "github.com/twistingmercury/mnemonic/internal/repository/graph"
	patternrepo "github.com/twistingmercury/mnemonic/internal/repository/pattern"
	openaisvc "github.com/twistingmercury/mnemonic/internal/service/openai"
)

// Service defines the worker-facing interface for processing enrichment jobs.
// The enrichment worker goroutine calls these methods; REST/MCP handlers do not.
type Service interface {
	// ClaimNextJob atomically claims the next pending enrichment job.
	// Returns (nil, nil) if no jobs are available.
	ClaimNextJob(ctx context.Context) (*enrichmentjob.Job, error)

	// ProcessJob runs the full enrichment pipeline for a claimed job:
	//   1. Load pattern from Postgres
	//   2. Generate embedding via EmbeddingService
	//   3. Store embedding in Postgres
	//   4. Extract concepts via ExtractionService
	//   5. Sync pattern node to Neo4j
	//   6. Sync concepts and MENTIONED_IN edges to Neo4j
	//   7. Sync agent associations to Neo4j
	//   8. Compute RELATED_TO edges
	//   9. Mark job completed and pattern enriched
	//
	// On failure at any pipeline step, marks the job as failed with error detail.
	// Returns nil when a step fails but the failure is recorded successfully.
	// Returns a non-nil error only for unrecoverable system errors (e.g., failure
	// to mark the job as failed or to update enrichment status).
	ProcessJob(ctx context.Context, job *enrichmentjob.Job) error

	// ReclaimStaleJobs resets jobs stuck in processing state back to pending.
	// Called periodically by the worker lifecycle manager.
	ReclaimStaleJobs(ctx context.Context) (int64, error)

	// CleanupCompletedJobs removes completed jobs older than the retention period.
	CleanupCompletedJobs(ctx context.Context) (int64, error)

	// CleanupFailedJobs removes failed jobs older than the retention period.
	CleanupFailedJobs(ctx context.Context) (int64, error)
}

// enrichmentService implements the Service interface.
type enrichmentService struct {
	jobRepo       enrichmentjob.Repository
	patternRepo   patternrepo.Repository
	agentRepo     agentrepo.Repository
	graphRepo     graphrepo.Repository
	embeddingSvc  openaisvc.EmbeddingService
	extractionSvc openaisvc.ExtractionService
	cfg           config.EnrichmentConfig
	logger        zerolog.Logger
}

// New creates a new enrichment Service backed by the given dependencies.
func New(
	jobRepo enrichmentjob.Repository,
	patternRepo patternrepo.Repository,
	agentRepo agentrepo.Repository,
	graphRepo graphrepo.Repository,
	embeddingSvc openaisvc.EmbeddingService,
	extractionSvc openaisvc.ExtractionService,
	cfg config.EnrichmentConfig,
	logger zerolog.Logger,
) Service {
	return &enrichmentService{
		jobRepo:       jobRepo,
		patternRepo:   patternRepo,
		agentRepo:     agentRepo,
		graphRepo:     graphRepo,
		embeddingSvc:  embeddingSvc,
		extractionSvc: extractionSvc,
		cfg:           cfg,
		logger:        logger,
	}
}

// ClaimNextJob atomically claims the next pending enrichment job.
// Returns (nil, nil) if no jobs are available.
func (s *enrichmentService) ClaimNextJob(ctx context.Context) (*enrichmentjob.Job, error) {
	return s.jobRepo.ClaimPending(ctx)
}

// ProcessJob runs the full enrichment pipeline for a claimed job.
func (s *enrichmentService) ProcessJob(ctx context.Context, job *enrichmentjob.Job) error {
	// Step 1: Load pattern from Postgres.
	pattern, err := s.patternRepo.Get(ctx, job.PatternID)
	if err != nil {
		return s.failJob(ctx, job, fmt.Errorf("load pattern: %w", err))
	}

	// Step 2: Generate embedding via OpenAI.
	embedding, err := s.embeddingSvc.Embed(ctx, pattern.Content)
	if err != nil {
		return s.failJob(ctx, job, fmt.Errorf("generate embedding: %w", err))
	}

	// Step 3: Store embedding in Postgres.
	if err := s.patternRepo.UpdateEmbedding(ctx, pattern.ID, embedding); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("store embedding: %w", err))
	}

	// Step 4: Extract concepts via OpenAI.
	concepts, err := s.extractionSvc.Extract(ctx, pattern.Content)
	if err != nil {
		return s.failJob(ctx, job, fmt.Errorf("extract concepts: %w", err))
	}

	// Step 5: Sync pattern node to Neo4j.
	graphPattern := &graphrepo.Pattern{
		ID:          pattern.ID,
		Name:        pattern.Name,
		Description: pattern.Description,
	}
	if err := s.graphRepo.SyncPattern(ctx, graphPattern); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("sync pattern to neo4j: %w", err))
	}

	// Step 6: Sync concepts and MENTIONED_IN edges to Neo4j.
	// Explicit mapping between openai.Concept and graphrepo.Concept is intentional:
	// the types are structurally similar but belong to different packages with
	// different responsibilities (extraction vs. graph storage).
	graphConcepts := make([]graphrepo.Concept, len(concepts))
	for i, c := range concepts {
		graphConcepts[i] = graphrepo.Concept{Name: c.Name, Type: c.Type}
	}
	if err := s.graphRepo.SyncConcepts(ctx, pattern.ID, graphConcepts); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("sync concepts to neo4j: %w", err))
	}

	// Step 7: Sync agent associations to Neo4j.
	// patternrepo.AgentAssociation has AgentID (UUID), but Neo4j Agent nodes are
	// keyed by name. Resolve agent names via agentRepo before syncing.
	graphAssocs, err := s.resolveAgentAssociations(ctx, pattern.ID)
	if err != nil {
		return s.failJob(ctx, job, fmt.Errorf("get agent associations: %w", err))
	}
	if err := s.graphRepo.SetPatternAgentRelevance(ctx, pattern.ID, graphAssocs); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("sync associations to neo4j: %w", err))
	}

	// Step 8: Compute RELATED_TO edges based on shared concepts.
	if err := s.graphRepo.ComputeRelatedToEdges(ctx, pattern.ID, s.cfg.RelatedToMinSimilarity); err != nil {
		return s.failJob(ctx, job, fmt.Errorf("compute related_to: %w", err))
	}

	// Step 9: Mark job completed (unrecoverable if this fails).
	if err := s.jobRepo.MarkCompleted(ctx, job.ID); err != nil {
		return fmt.Errorf("mark job completed: %w", err)
	}

	// Step 10: Update pattern enrichment status (unrecoverable if this fails).
	if err := s.patternRepo.UpdateEnrichmentStatus(ctx, pattern.ID, "enriched", nil); err != nil {
		return fmt.Errorf("update enrichment status: %w", err)
	}

	return nil
}

// resolveAgentAssociations loads pattern-agent associations from Postgres and
// resolves agent UUIDs to names for Neo4j sync. Agents that cannot be resolved
// are logged and skipped.
func (s *enrichmentService) resolveAgentAssociations(ctx context.Context, patternID uuid.UUID) ([]graphrepo.AgentAssociation, error) {
	associations, err := s.patternRepo.GetAgentAssociations(ctx, patternID)
	if err != nil {
		return nil, err
	}

	graphAssocs := make([]graphrepo.AgentAssociation, 0, len(associations))
	for _, a := range associations {
		agent, err := s.agentRepo.GetByID(ctx, a.AgentID)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Str("agent_id", a.AgentID.String()).
				Msg("skip association: agent not found")
			continue
		}
		graphAssocs = append(graphAssocs, graphrepo.AgentAssociation{
			AgentName: agent.Name,
			Relevance: a.Relevance,
		})
	}

	return graphAssocs, nil
}

// failJob marks the job as failed and updates the pattern enrichment status.
// Returns nil on success (the failure is recorded). Returns an error only when
// the failure itself cannot be recorded (unrecoverable).
func (s *enrichmentService) failJob(ctx context.Context, job *enrichmentjob.Job, cause error) error {
	if err := s.jobRepo.MarkFailed(ctx, job.ID, cause, s.cfg.RetryDelay); err != nil {
		s.logger.Error().
			Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to mark job as failed")
		return fmt.Errorf("mark job failed: %w (original cause: %v)", err, cause)
	}

	errMsg := cause.Error()
	if err := s.patternRepo.UpdateEnrichmentStatus(ctx, job.PatternID, "failed", &errMsg); err != nil {
		s.logger.Error().
			Err(err).
			Str("pattern_id", job.PatternID.String()).
			Msg("failed to update pattern enrichment status")
		return fmt.Errorf("update enrichment status: %w (original cause: %v)", err, cause)
	}

	return nil
}

// ReclaimStaleJobs resets jobs stuck in processing state back to pending.
func (s *enrichmentService) ReclaimStaleJobs(ctx context.Context) (int64, error) {
	count, err := s.jobRepo.ReclaimStale(ctx, s.cfg.JobTimeout)
	if err != nil {
		return 0, fmt.Errorf("reclaim stale jobs: %w", err)
	}
	return count, nil
}

// CleanupCompletedJobs removes completed jobs older than the retention period.
func (s *enrichmentService) CleanupCompletedJobs(ctx context.Context) (int64, error) {
	count, err := s.jobRepo.DeleteCompleted(ctx, s.cfg.CompletedRetention)
	if err != nil {
		return 0, fmt.Errorf("cleanup completed jobs: %w", err)
	}
	return count, nil
}

// CleanupFailedJobs removes failed jobs older than the retention period.
func (s *enrichmentService) CleanupFailedJobs(ctx context.Context) (int64, error) {
	count, err := s.jobRepo.DeleteFailed(ctx, s.cfg.FailedRetention)
	if err != nil {
		return 0, fmt.Errorf("cleanup failed jobs: %w", err)
	}
	return count, nil
}

// Compile-time check that *enrichmentService implements Service.
var _ Service = (*enrichmentService)(nil)

// Compile-time check that New returns the Service interface.
var _ func(
	enrichmentjob.Repository,
	patternrepo.Repository,
	agentrepo.Repository,
	graphrepo.Repository,
	openaisvc.EmbeddingService,
	openaisvc.ExtractionService,
	config.EnrichmentConfig,
	zerolog.Logger,
) Service = New
