package enricher

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/twistingmercury/mnemonic/internal/config"
	enrichmentsvc "github.com/twistingmercury/mnemonic/internal/service/enrichment"
)

// defaultMaintenanceInterval is the interval between maintenance loop iterations.
// Maintenance reclaims stale jobs and cleans up old completed/failed jobs.
const defaultMaintenanceInterval = 5 * time.Minute

// Worker polls for pending enrichment jobs and processes them using
// the EnrichmentService. It runs a configurable number of concurrent claim-process
// goroutines plus a single maintenance goroutine.
type Worker struct {
	svc    enrichmentsvc.Service
	cfg    config.EnrichmentConfig
	logger zerolog.Logger
}

// New creates a Worker that processes enrichment jobs using svc, configured by
// cfg. The logger is used for structured logging of worker lifecycle events and
// errors.
func New(svc enrichmentsvc.Service, cfg config.EnrichmentConfig, logger zerolog.Logger) *Worker {
	return &Worker{
		svc:    svc,
		cfg:    cfg,
		logger: logger.With().Str("component", "enrichment_worker").Logger(),
	}
}

// Run starts the worker pool and blocks until ctx is cancelled. It launches
// cfg.WorkerCount claim-process goroutines plus one maintenance goroutine.
// All goroutines are coordinated via an errgroup; Run returns nil on graceful
// shutdown (context cancellation).
func (w *Worker) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// Start claim-process workers.
	for i := range w.cfg.WorkerCount {
		workerID := i
		g.Go(func() error {
			w.runWorker(ctx, workerID)
			return nil
		})
	}

	// Start maintenance goroutine.
	g.Go(func() error {
		w.runMaintenance(ctx)
		return nil
	})

	w.logger.Info().
		Int("worker_count", w.cfg.WorkerCount).
		Dur("poll_interval", w.cfg.PollInterval).
		Msg("enrichment worker started")

	err := g.Wait()

	w.logger.Info().Msg("enrichment worker stopped")
	return err
}

// runWorker is the claim-process loop for a single worker goroutine. It
// repeatedly claims and processes jobs until ctx is cancelled.
func (w *Worker) runWorker(ctx context.Context, id int) {
	log := w.logger.With().Int("worker_id", id).Logger()
	log.Debug().Msg("worker goroutine started")

	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("worker goroutine stopping")
			return
		default:
		}

		job, err := w.svc.ClaimNextJob(ctx)
		if err != nil {
			// Check for context cancellation to avoid noisy logging on shutdown.
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).Msg("failed to claim job")
			w.sleep(ctx, w.cfg.PollInterval)
			continue
		}

		if job == nil {
			// No pending jobs; sleep before polling again.
			w.sleep(ctx, w.cfg.PollInterval)
			continue
		}

		log.Info().
			Str("job_id", job.ID.String()).
			Str("pattern_id", job.PatternID.String()).
			Msg("processing enrichment job")

		if err := w.svc.ProcessJob(ctx, job); err != nil {
			// Non-nil error from ProcessJob means the failure could not be
			// recorded (unrecoverable). Log at error level.
			log.Error().
				Err(err).
				Str("job_id", job.ID.String()).
				Msg("enrichment job failed with unrecoverable error")
		} else {
			log.Info().
				Str("job_id", job.ID.String()).
				Msg("enrichment job completed")
		}
	}
}

// runMaintenance periodically reclaims stale jobs and cleans up old
// completed/failed jobs. It runs until ctx is cancelled.
func (w *Worker) runMaintenance(ctx context.Context) {
	log := w.logger.With().Str("loop", "maintenance").Logger()
	log.Debug().Msg("maintenance goroutine started")

	ticker := time.NewTicker(defaultMaintenanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("maintenance goroutine stopping")
			return
		case <-ticker.C:
			w.doMaintenance(ctx, log)
		}
	}
}

// doMaintenance performs a single maintenance cycle: reclaim stale jobs,
// cleanup completed jobs, and cleanup failed jobs.
func (w *Worker) doMaintenance(ctx context.Context, log zerolog.Logger) {
	if ctx.Err() != nil {
		return
	}

	reclaimed, err := w.svc.ReclaimStaleJobs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to reclaim stale jobs")
	} else if reclaimed > 0 {
		log.Info().Int64("count", reclaimed).Msg("reclaimed stale jobs")
	}

	completedCleaned, err := w.svc.CleanupCompletedJobs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to cleanup completed jobs")
	} else if completedCleaned > 0 {
		log.Info().Int64("count", completedCleaned).Msg("cleaned up completed jobs")
	}

	failedCleaned, err := w.svc.CleanupFailedJobs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to cleanup failed jobs")
	} else if failedCleaned > 0 {
		log.Info().Int64("count", failedCleaned).Msg("cleaned up failed jobs")
	}
}

// sleep blocks for the given duration or until ctx is cancelled, whichever
// comes first.
func (w *Worker) sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
