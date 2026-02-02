---
entity_name: Branch Strategy Guideline
entity_type: engineering-guideline
language: agnostic
domain: source-management
description: Trunk-based development with short-lived feature branches
tags:
  - git
  - branching
  - trunk-based
  - source-control
---

# Branch Strategy Guideline

## Overview

We use trunk-based development with one stable main branch and short-lived feature branches. This strategy ensures stable production-ready code while enabling rapid integration and deployment.

## Rules

### Main Branch (Trunk)

**MUST:**

- Maintain main branch as the single source of truth for production-ready code
- Protect main branch with:
  - Required pull request reviews
  - All CI checks must pass
  - No force pushes
  - Branches must be up to date before merging
- Ensure every merge to main triggers automated Docker image build and push

**MUST NOT:**

- Commit directly to main branch (branch protection enforces this)
- Merge broken or untested code to main

### Development Branches

**MUST:**

- Create branches from main for all new work
- Keep branches short-lived: maximum 2 working days
- Focus each branch on one task only
- Delete branches after successful merge and production deployment
- Merge changes from main into your branch at least once daily
- Merge from main before requesting code review

**MUST NOT:**

- Keep branches alive longer than 2 working days without team lead approval
- Work on multiple unrelated tasks in one branch
- Let old branches accumulate after merge

### Branch Naming Convention

**MUST:**

- Use `feature/<jira-ticket-number>` for new features
  - Example: `feature/ex-1234`
- Use `bug/<jira-ticket-number>` for bug fixes
  - Example: `bug/ex-1235`

**MUST NOT:**

- Use arbitrary or inconsistent branch names
- Omit the Jira ticket reference

### Pull Requests

**MUST:**

- Create pull request for all changes to main
- Pass all CI checks before merge
- Obtain approval from another team member
- Document justification in PR description if branch exceeds 2 days (requires team lead approval)
- Ensure code is deployment-ready when merging to main

**MUST NOT:**

- Bypass code review process
- Merge without CI checks passing
- Merge without required approvals

### Deployment Readiness

**MUST:**

- Ensure branches can merge and deploy independently (handle edge cases as needed)
- Tag codebase with version number after merge when ready for production deployment
- Consider code production-ready at time of merge to main

**MUST NOT:**

- Merge code that isn't ready for production deployment
- Skip version tagging for production releases

## Rationale

Trunk-based development with short-lived branches:

- Reduces merge conflicts through frequent integration
- Enables rapid feedback through continuous integration
- Maintains deployment readiness of main branch
- Simplifies release management through automated builds
- Encourages small, focused changes that are easier to review
- Prevents long-running branches that diverge from main

## Related Guidelines

- Code Review Guidelines
- CI/CD Pipeline Guidelines
- Version Tagging Guidelines
- Pull Request Guidelines

## References

- [Trunk Based Development](https://trunkbaseddevelopment.com/)
- [Git Basics - Tagging](https://git-scm.com/book/en/v2/Git-Basics-Tagging)
