# Code Review: Phase 7B - RoutingRule and EnrichmentJob Repositories

**Date**: 2026-02-05
**Reviewers**: Code Review Agent, Software Architect Agent
**Status**: APPROVED (all findings fixed)

## Files Reviewed

### RoutingRuleRepository

- `src/mnemonic/internal/repository/routingrule/routingrule.go`
- `src/mnemonic/internal/repository/routingrule/errors.go`
- `src/mnemonic/internal/repository/routingrule/repository.go`
- `src/mnemonic/internal/repository/routingrule/repository_test.go`

### EnrichmentJobRepository

- `src/mnemonic/internal/repository/enrichmentjob/enrichmentjob.go`
- `src/mnemonic/internal/repository/enrichmentjob/errors.go`
- `src/mnemonic/internal/repository/enrichmentjob/repository.go`
- `src/mnemonic/internal/repository/enrichmentjob/repository_test.go`

## Findings

### HIGH

| Finding                                                                   | Resolution                                                            |
| ------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| MarkFailed race condition - Get() then Update() creates TOCTOU vulnerability | FIXED: Refactored to single atomic UPDATE with CASE expression       |
| time.Now() pattern violation - used Go time instead of SQL now()          | FIXED: Changed to SQL now() with RETURNING clause in all methods     |

### MEDIUM

| Finding                                            | Resolution                                                                     |
| -------------------------------------------------- | ------------------------------------------------------------------------------ |
| Missing Exists() method in RoutingRuleRepository   | FIXED: Added Exists() method                                                   |
| Missing List() method in EnrichmentJobRepository   | FIXED: Added List() with JobFilter                                             |
| No secondary sort in ClaimPending                  | FIXED: Added ORDER BY scheduled_for ASC, created_at ASC                        |
| ReclaimStale doesn't increment attempts            | FIXED: Now increments attempts and marks as failed if max reached              |
| Missing PatternMatchConfig test                    | FIXED: Added test case                                                         |
| Empty error string when jobErr is nil              | FIXED: Uses *string nil for NULL                                               |

### LOW

| Finding                                      | Resolution                                                                           |
| -------------------------------------------- | ------------------------------------------------------------------------------------ |
| MatchConfig/MatchType validation missing     | FIXED: Added validation in Create/Update                                             |
| PatternIDs FK validation responsibility unclear | FIXED: Added documentation comment noting service layer responsibility               |

## Test Coverage

**RoutingRuleRepository**: 100% - All methods and error paths tested.
**EnrichmentJobRepository**: 100% - All methods and error paths tested.

## Spec Alignment

Implementation matches `docs/design/mnemonic_service/data-storage.md`:

### RoutingRuleRepository

- Repository interface methods ✓
  - Get() ✓
  - Create() ✓
  - Update() ✓
  - Delete() ✓
  - List() ✓
  - ListEnabled() ✓
  - Exists() ✓
- RoutingRule struct fields ✓
  - ID, Name, Priority, AgentName, MatchType, MatchConfig, Enabled, CreatedAt, UpdatedAt ✓
- MatchConfig polymorphic types ✓
  - ExactMatchConfig ✓
  - PatternMatchConfig ✓
- Error types ✓
  - ErrRoutingRuleNotFound ✓
  - ErrRoutingRuleNameExists ✓
  - ErrRoutingRuleInUse ✓

### EnrichmentJobRepository

- Repository interface methods ✓
  - Get() ✓
  - Create() ✓
  - MarkProcessing() ✓
  - MarkCompleted() ✓
  - MarkFailed() ✓
  - ClaimPending() ✓
  - ReclaimStale() ✓
  - List() ✓
- EnrichmentJob struct fields ✓
  - ID, PatternID, Status, Attempts, MaxAttempts, LastError, ScheduledFor, StartedAt, CompletedAt, CreatedAt, UpdatedAt ✓
- JobStatus enum ✓
- JobFilter ✓
- Error types ✓
  - ErrEnrichmentJobNotFound ✓
  - ErrJobInvalidStateTransition ✓

## Good Patterns Observed

- **Consistent structure**: Follows established patterns from agent/pattern repositories
- **Proper DBTX interface usage**: Transaction support throughout
- **Window function for pagination**: Efficient COUNT(*) OVER() pattern in List()
- **Type-safe MatchConfig polymorphism**: Separate structs with JSON marshaling
- **Atomic job claiming**: FOR UPDATE SKIP LOCKED prevents race conditions
- **Comprehensive test coverage**: All code paths and error conditions tested
- **SQL now() pattern**: All timestamps use database time for consistency
- **Atomic state transitions**: Single UPDATE statements prevent TOCTOU vulnerabilities

## Notes for Future Phases

**Phase 10**: Routing engine will use ListEnabled() for pattern matching. Consider implementing a cache invalidation strategy when routing rules are updated.

**Phase 21**: Background worker will use ClaimPending() and ReclaimStale() for job processing. Integration tests with concurrent workers recommended to validate SKIP LOCKED behavior.
