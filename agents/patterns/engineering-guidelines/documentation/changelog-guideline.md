---
entity_name: CHANGELOG Guideline
entity_type: engineering-guideline
language: agnostic
domain: documentation
description: Keep a Changelog format requirements for version tracking
tags:
  - documentation
  - changelog
  - versioning
  - releases
---

# CHANGELOG Guideline

## Purpose

CHANGELOG.md documents what changed between versions of a project, helping users understand the impact of upgrades and maintainers track project history.

## When to Use CHANGELOG.md

**Required for:**

- Projects with semantic version releases (v1.0.0, v1.2.3, etc.)
- Production-ready applications and services
- Libraries and packages distributed to users

**Not required for:**

- Projects without versioned releases (internal scripts, one-off tools)
- These may use commit history instead of maintaining a formal CHANGELOG

## Format Standard

All CHANGELOG.md files MUST follow the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format.

## Required Structure

### File Header

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
```

### Unreleased Section

Every CHANGELOG.md MUST maintain an `## [Unreleased]` section at the top for changes that haven't been released yet:

```markdown
## [Unreleased]

### Added
- New feature descriptions here

### Changed
- Modifications to existing functionality

### Deprecated
- Features marked for removal in future versions

### Removed
- Features removed in this version

### Fixed
- Bug fixes

### Security
- Security vulnerability fixes and improvements
```

### Version Sections

Each released version gets its own section with version number and release date:

```markdown
## [1.2.0] - 2026-01-15

### Added
- User authentication with OAuth2 support
- API rate limiting middleware

### Changed
- Updated dependency versions for security patches
- Improved error messages in validation layer

### Fixed
- Database connection pool exhaustion under high load
```

## Change Categories

Group changes under these standardized headings:

- **Added**: New features or capabilities
- **Changed**: Changes to existing functionality
- **Deprecated**: Features marked for removal (still working but discouraged)
- **Removed**: Features that have been removed
- **Fixed**: Bug fixes
- **Security**: Security vulnerability fixes and security improvements

## What to Document

**Focus on user-impacting changes:**

- New features users can leverage
- Breaking changes that require code updates
- Bug fixes that change behavior
- Security fixes (without exposing vulnerability details)
- Deprecated features with migration guidance

**Skip internal implementation details:**

- Refactoring that doesn't change behavior
- Internal code reorganization
- Test improvements (unless it impacts users)
- Development tooling changes

## Writing Style

**Good changelog entries:**

- "Added OAuth2 authentication support for enterprise SSO integration"
- "Fixed race condition causing duplicate transaction processing"
- "Deprecated legacy REST API endpoints (use GraphQL API instead)"

**Poor changelog entries:**

- "Updated files" (too vague)
- "Refactored database layer to use repository pattern" (internal detail)
- "Fixed bug" (not specific enough)

## Release Preparation Process

When preparing a new release:

1. **Review Unreleased section**: Ensure all changes since last release are documented
2. **Create version section**: Add new version header with release date
3. **Move changes**: Transfer all items from `[Unreleased]` to the new version section
4. **Clear Unreleased**: Leave `[Unreleased]` section empty but present
5. **Update version links**: Add comparison links at bottom of file (if using)

### Example Release Transformation

**Before release:**

```markdown
## [Unreleased]

### Added
- User profile management API
- Email notification system

### Fixed
- Login timeout issue
```

**After releasing v1.3.0:**

```markdown
## [Unreleased]

## [1.3.0] - 2026-02-02

### Added
- User profile management API
- Email notification system

### Fixed
- Login timeout issue
```

## Integration with Versioning

CHANGELOG.md works alongside semantic versioning:

- **Major version (1.0.0)**: Typically includes "Removed" or breaking "Changed" items
- **Minor version (0.1.0)**: Typically includes "Added" or non-breaking "Changed" items
- **Patch version (0.0.1)**: Typically includes "Fixed" or "Security" items

## Audience

**Primary users:**

- Users upgrading between versions
- Maintainers reviewing project history
- Operations teams planning deployments

**Target clarity:**

- Write for users who need to know impact, not developers who made changes
- Explain what changed and why users care
- Include migration guidance for breaking changes

## Enforcement

CHANGELOG.md updates are part of the [Definition of Done](./deliver-solutions-that-work.md#definition-of-done):

- CHANGELOG.md MUST be updated before release
- Changes MUST be documented in appropriate category
- Unreleased section MUST be maintained between releases
- Format MUST follow Keep a Changelog standard

## References

- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
- [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
- [Engineering Handbook: Versioning Our Solutions](./versioning-our-solutions.md)
- [Engineering Handbook: Importance of Documentation](./importance-of-documentation.md)
