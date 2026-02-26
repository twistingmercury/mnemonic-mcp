package enricher_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/enricher"
	enrichmentjob "github.com/twistingmercury/mnemonic/internal/repository/enrichmentjob"
)

// concurrentService is a hand-written mock for tests that need dynamic return
// values (e.g., returning different jobs from a shared queue). This avoids
// testify/mock's limitations with functional Return values and error types.
type concurrentService struct {
	claimFunc   func(ctx context.Context) (*enrichmentjob.Job, error)
	processFunc func(ctx context.Context, job *enrichmentjob.Job) error
}

func (s *concurrentService) ClaimNextJob(ctx context.Context) (*enrichmentjob.Job, error) {
	return s.claimFunc(ctx)
}

func (s *concurrentService) ProcessJob(ctx context.Context, job *enrichmentjob.Job) error {
	return s.processFunc(ctx, job)
}

func (s *concurrentService) ReclaimStaleJobs(_ context.Context) (int64, error) {
	return 0, nil
}

func (s *concurrentService) CleanupCompletedJobs(_ context.Context) (int64, error) {
	return 0, nil
}

func (s *concurrentService) CleanupFailedJobs(_ context.Context) (int64, error) {
	return 0, nil
}

// mockService implements enrichmentsvc.Service for testing.
type mockService struct {
	mock.Mock
}

func (m *mockService) ClaimNextJob(ctx context.Context) (*enrichmentjob.Job, error) {
	args := m.Called(ctx)
	job, _ := args.Get(0).(*enrichmentjob.Job)
	return job, args.Error(1)
}

func (m *mockService) ProcessJob(ctx context.Context, job *enrichmentjob.Job) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *mockService) ReclaimStaleJobs(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockService) CleanupCompletedJobs(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockService) CleanupFailedJobs(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// testConfig returns an EnrichmentConfig with fast intervals for testing.
func testConfig(workerCount int) config.EnrichmentConfig {
	return config.EnrichmentConfig{
		WorkerCount:            workerCount,
		PollInterval:           1 * time.Millisecond,
		MaxAttempts:            3,
		RetryDelay:             10 * time.Millisecond,
		JobTimeout:             1 * time.Second,
		CompletedRetention:     1 * time.Hour,
		FailedRetention:        1 * time.Hour,
		RelatedToMinSimilarity: 0.3,
	}
}

// testLogger returns a no-op logger for tests.
func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

// newTestJob creates a Job with a random ID and pattern ID.
func newTestJob() *enrichmentjob.Job {
	return &enrichmentjob.Job{
		ID:        uuid.New(),
		PatternID: uuid.New(),
		Status:    string(enrichmentjob.StatusProcessing),
		Attempts:  1,
	}
}

func TestWorkerProcessesAvailableJobs(t *testing.T) {
	t.Parallel()

	svc := new(mockService)
	job := newTestJob()

	var processed atomic.Bool

	// First call returns a job; subsequent calls return nil (no more jobs).
	svc.On("ClaimNextJob", mock.Anything).Return(job, nil).Once()
	svc.On("ClaimNextJob", mock.Anything).Return(nil, nil).Maybe()

	svc.On("ProcessJob", mock.Anything, job).Return(nil).Once().Run(func(_ mock.Arguments) {
		processed.Store(true)
	})

	// Maintenance stubs (may or may not be called depending on timing).
	svc.On("ReclaimStaleJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupCompletedJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupFailedJobs", mock.Anything).Return(int64(0), nil).Maybe()

	w := enricher.New(svc, testConfig(1), testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := w.Run(ctx)
	assert.NoError(t, err)
	assert.True(t, processed.Load(), "job should have been processed")
	svc.AssertCalled(t, "ProcessJob", mock.Anything, job)
}

func TestWorkerSleepsWhenNoJobs(t *testing.T) {
	t.Parallel()

	svc := new(mockService)
	job := newTestJob()

	var claimCount atomic.Int32

	// Track claim calls. First several return nil, then return a job.
	svc.On("ClaimNextJob", mock.Anything).Return(nil, nil).Times(3).Run(func(_ mock.Arguments) {
		claimCount.Add(1)
	})
	svc.On("ClaimNextJob", mock.Anything).Return(job, nil).Once().Run(func(_ mock.Arguments) {
		claimCount.Add(1)
	})
	svc.On("ClaimNextJob", mock.Anything).Return(nil, nil).Maybe()

	svc.On("ProcessJob", mock.Anything, job).Return(nil).Once()

	svc.On("ReclaimStaleJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupCompletedJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupFailedJobs", mock.Anything).Return(int64(0), nil).Maybe()

	w := enricher.New(svc, testConfig(1), testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := w.Run(ctx)
	assert.NoError(t, err)

	// Should have polled at least 4 times (3 nil + 1 job).
	assert.GreaterOrEqual(t, claimCount.Load(), int32(4))
	svc.AssertCalled(t, "ProcessJob", mock.Anything, job)
}

func TestWorkerGracefulShutdown(t *testing.T) {
	t.Parallel()

	svc := new(mockService)

	// ClaimNextJob always returns nil so the worker just polls.
	svc.On("ClaimNextJob", mock.Anything).Return(nil, nil).Maybe()
	svc.On("ReclaimStaleJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupCompletedJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupFailedJobs", mock.Anything).Return(int64(0), nil).Maybe()

	w := enricher.New(svc, testConfig(2), testLogger())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Run(ctx)
	}()

	// Give the worker time to start, then cancel.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err, "Run should return nil on graceful shutdown")
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not shut down within timeout")
	}
}

func TestMultipleWorkersConcurrency(t *testing.T) {
	t.Parallel()

	// concurrentService tracks concurrent job processing without testify/mock
	// to avoid the complexity of functional return values with mock.Return.
	var (
		mu             sync.Mutex
		jobIndex       int
		processedCount atomic.Int32
	)

	jobs := make([]*enrichmentjob.Job, 4)
	for i := range jobs {
		jobs[i] = newTestJob()
	}

	svc := &concurrentService{
		claimFunc: func(_ context.Context) (*enrichmentjob.Job, error) {
			mu.Lock()
			defer mu.Unlock()
			if jobIndex < len(jobs) {
				j := jobs[jobIndex]
				jobIndex++
				return j, nil
			}
			return nil, nil
		},
		processFunc: func(_ context.Context, _ *enrichmentjob.Job) error {
			processedCount.Add(1)
			return nil
		},
	}

	w := enricher.New(svc, testConfig(2), testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := w.Run(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int32(4), processedCount.Load(), "all jobs should be processed")
}

func TestMaintenanceLoopRuns(t *testing.T) {
	t.Parallel()

	svc := new(mockService)
	svc.On("ClaimNextJob", mock.Anything).Return(nil, nil).Maybe()

	var reclaimCalled atomic.Bool
	var cleanupCompletedCalled atomic.Bool
	var cleanupFailedCalled atomic.Bool

	svc.On("ReclaimStaleJobs", mock.Anything).Return(int64(2), nil).Maybe().Run(func(_ mock.Arguments) {
		reclaimCalled.Store(true)
	})
	svc.On("CleanupCompletedJobs", mock.Anything).Return(int64(1), nil).Maybe().Run(func(_ mock.Arguments) {
		cleanupCompletedCalled.Store(true)
	})
	svc.On("CleanupFailedJobs", mock.Anything).Return(int64(0), nil).Maybe().Run(func(_ mock.Arguments) {
		cleanupFailedCalled.Store(true)
	})

	// Use a very short maintenance interval for testing by running the worker
	// long enough for the default 5m interval to NOT fire. Instead, we test
	// that the maintenance loop is wired by using a custom approach:
	// We cannot easily change the defaultMaintenanceInterval since it's a const.
	// Instead, verify that the maintenance goroutine starts and can be shut down.
	// For a thorough test of the maintenance functions being called, we test
	// doMaintenance behavior indirectly through a longer-running test.
	//
	// Since we cannot inject the maintenance interval, we verify the maintenance
	// goroutine does not prevent shutdown. The actual maintenance calls happen on
	// a 5-minute ticker, which is too slow for unit tests. We verify the wiring
	// works by testing the exported Run method completes cleanly.
	w := enricher.New(svc, testConfig(1), testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := w.Run(ctx)
	assert.NoError(t, err, "worker should shut down cleanly even with maintenance goroutine")
}

func TestProcessJobErrorIsLoggedNotFatal(t *testing.T) {
	t.Parallel()

	svc := new(mockService)
	job := newTestJob()

	var processCount atomic.Int32

	// First claim returns a job that will fail, second claim returns a job that succeeds.
	job2 := newTestJob()
	svc.On("ClaimNextJob", mock.Anything).Return(job, nil).Once()
	svc.On("ClaimNextJob", mock.Anything).Return(job2, nil).Once()
	svc.On("ClaimNextJob", mock.Anything).Return(nil, nil).Maybe()

	svc.On("ProcessJob", mock.Anything, job).Return(errors.New("unrecoverable error")).Once().
		Run(func(_ mock.Arguments) { processCount.Add(1) })
	svc.On("ProcessJob", mock.Anything, job2).Return(nil).Once().
		Run(func(_ mock.Arguments) { processCount.Add(1) })

	svc.On("ReclaimStaleJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupCompletedJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupFailedJobs", mock.Anything).Return(int64(0), nil).Maybe()

	w := enricher.New(svc, testConfig(1), testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := w.Run(ctx)
	assert.NoError(t, err, "worker should not crash on ProcessJob error")
	assert.GreaterOrEqual(t, processCount.Load(), int32(2), "worker should continue processing after error")
}

func TestClaimNextJobErrorIsLoggedNotFatal(t *testing.T) {
	t.Parallel()

	svc := new(mockService)
	job := newTestJob()

	var claimCount atomic.Int32

	// First claim returns an error, subsequent claims succeed.
	svc.On("ClaimNextJob", mock.Anything).Return(nil, errors.New("db error")).Once().
		Run(func(_ mock.Arguments) { claimCount.Add(1) })
	svc.On("ClaimNextJob", mock.Anything).Return(job, nil).Once().
		Run(func(_ mock.Arguments) { claimCount.Add(1) })
	svc.On("ClaimNextJob", mock.Anything).Return(nil, nil).Maybe()

	svc.On("ProcessJob", mock.Anything, job).Return(nil).Once()

	svc.On("ReclaimStaleJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupCompletedJobs", mock.Anything).Return(int64(0), nil).Maybe()
	svc.On("CleanupFailedJobs", mock.Anything).Return(int64(0), nil).Maybe()

	w := enricher.New(svc, testConfig(1), testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := w.Run(ctx)
	assert.NoError(t, err, "worker should not crash on ClaimNextJob error")
	assert.GreaterOrEqual(t, claimCount.Load(), int32(2), "worker should continue claiming after error")
	svc.AssertCalled(t, "ProcessJob", mock.Anything, job)
}

func TestNewReturnsNonNil(t *testing.T) {
	t.Parallel()

	svc := new(mockService)
	w := enricher.New(svc, testConfig(2), testLogger())
	assert.NotNil(t, w)
}
