---
entity_name: Test Coverage Thresholds Guideline
entity_type: engineering-guideline
language: agnostic
domain: testing
description: Minimum code coverage requirements for unit and integration tests
tags:
  - testing
  - coverage
  - quality
  - ci-cd
---

# Test Coverage Thresholds Guideline

## Overview

This guideline defines minimum code coverage requirements for unit and integration tests, along with testing philosophies for different test types.

## Coverage Requirements

### Minimum Thresholds

Apply these minimum line coverage targets across the codebase:

**75% minimum overall line coverage**

- This is the baseline target for the entire codebase.
- If a file has 20% coverage but the other 80% is just constants and type definitions, that's acceptable.
- Most testing frameworks allow exclusion of constants, type definitions, and similar code from coverage reports - use these features.

**95% for critical paths**

Apply this higher threshold to:

- Authentication and authorization logic
- Payment processing code
- Data integrity operations
- Security-sensitive code

### Documenting Exceptions

If you cannot hit these coverage targets, document why:

- Legacy code being replaced
- Integration code better tested at the E2E level
- Generated code or boilerplate
- Code excluded by framework configuration

Include the reasoning in code comments or test documentation so future developers understand the decision.

## What Not to Test

Skip tests that don't provide value:

- Constants and configuration values
- Type definitions and data structures
- Generated code (unless the generator is custom)
- Simple getters and setters with no logic
- Code that is better validated through E2E tests

Document why you skipped testing specific code so the reasoning is clear to future maintainers.

## End-to-End Testing Philosophy

### Black-Box Testing Approach

E2E tests verify the system works as users or API clients expect:

**Test from the consumer perspective**

- Don't import or depend on your service's code.
- Test it like a black box, the way a real user would.
- Use curl for APIs, run CLI tools directly, interact through public interfaces only.
- Write tests in whatever language makes sense, but treat the system under test as external.

**Test based on documentation**

- Tests should rely only on what a client would have access to, such as API documentation.
- If it's not in the public documentation, it shouldn't be in the E2E tests.

### E2E Test Flow

1. **Execute the action** - Perform the operation a real user would do
2. **Verify the response** - Check you got what you expected (test all response codes: 200s, 400s, 500s)
3. **Check state changes** - If the operation should have modified data, verify it in the database or storage

### E2E Test Priority

Prioritize testing efforts in this order:

1. All success paths
2. Documented error responses
3. Authentication and authorization boundaries
4. Edge cases

Use judgment on testing every permutation of query parameters or request variations.

### E2E Test Infrastructure

- Tests run in isolated Docker environments
- Spin up everything the service needs: databases, message queues, dependent services
- Tests run automatically in CI
- If E2E tests fail, the build fails

## Documentation Testing

Test that documentation is accurate and usable.

### Gold Standard

Someone who joined the team last week should be able to follow your docs and succeed without asking for help.

### Why Test Documentation

- Wrong or confusing documentation wastes hours or days
- You're writing for the team member who'll maintain this code six months from now
- Future you who's forgotten everything will thank past you

### What to Test

Verify that:

- Setup instructions actually work
- Build commands execute successfully
- Deployment steps produce working systems
- Troubleshooting guides resolve actual issues
- Code examples compile and run

## CI Pipeline Requirements

Code must meet these coverage and testing standards before merging:

- CI pipeline is green (all unit and E2E tests pass)
- Coverage thresholds are met or exceptions are documented
- Tests run automatically in the build pipeline
- If tests fail, the build fails

## Related Guidelines

- [Unit Testing Best Practices](./unit-testing-guideline.md)
- [E2E Test Infrastructure](../ci-cd/e2e-test-setup-guideline.md)
- [Definition of Done](../quality/definition-of-done-guideline.md)
