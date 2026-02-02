---
entity_name: Branch Protection Guideline
entity_type: engineering-guideline
language: agnostic
domain: source-management
description: Protected branch rules for main/develop branches
tags:
  - git
  - branch-protection
  - code-review
  - security
---

# Branch Protection Guideline

## Purpose

Branch protection rules prevent accidental or unauthorized changes to critical branches (main/master/develop) and enforce quality gates through automated checks and peer review.

## Protected Branches

Apply these rules to:

- `main` (or `master`)
- `develop` (if using git-flow or similar)
- Release branches (if applicable)

## Required Protection Rules

### 1. No Direct Commits

**Rule**: Block direct pushes to protected branches.

**Rationale**: All changes must go through pull requests to ensure review and quality gates.

**Implementation**:

- Configure branch protection to require pull requests for all changes
- Remove direct push access for all users, including administrators (with emergency bypass option)

### 2. Pull Request Required

**Rule**: All changes must be submitted via pull request.

**Rationale**: Enables code review, automated testing, and documentation of change rationale.

**Implementation**:

- Require pull requests before merging
- Enforce PR template usage for consistent change documentation
- Require descriptive PR titles and descriptions

### 3. Code Review Approval Required

**Rule**: Pull requests require approval from at least one other team member before merge.

**Rationale**: Peer review catches bugs, security issues, and ensures knowledge sharing.

**Implementation**:

- Minimum 1 approval required
- Cannot approve your own pull requests
- Dismiss stale approvals when new commits are pushed
- Require review from code owners for specific paths (if applicable)

### 4. All CI Checks Must Pass

**Rule**: All continuous integration status checks must pass before merge.

**Rationale**: Automated tests and scans verify code quality, security, and functionality.

**Implementation**:

- Require status checks to pass before merging
- No exceptions - if CI is red, PR is blocked
- Include these checks:
  - Unit tests
  - Integration tests
  - End-to-end tests (if applicable)
  - Linting and code formatting
  - Security vulnerability scanning
  - License compliance checking

### 5. No Force Pushes

**Rule**: Force pushes to protected branches are blocked.

**Rationale**: Prevents history rewriting that can cause loss of work and confusion.

**Implementation**:

- Block force pushes for all users
- Block deletion of protected branches
- Use `git revert` for undoing changes instead of rewriting history

### 6. Branch Must Be Up-to-Date Before Merge

**Rule**: Feature branch must be current with target branch before merging.

**Rationale**: Ensures changes are tested against latest code and prevents integration issues.

**Implementation**:

- Require branches to be up-to-date before merging
- Merge or rebase target branch into feature branch before final approval
- Re-run CI checks after updating to verify compatibility

## Workflow Impact

### Developer Workflow

1. Create feature branch from protected branch
2. Make changes and commit to feature branch
3. Push feature branch to remote
4. Create pull request to protected branch
5. Address code review feedback
6. Ensure CI checks pass
7. Update branch with latest changes from target
8. Obtain approval from reviewer
9. Merge pull request

### Emergency Procedures

In rare emergency situations (production outage, critical security fix):

1. Follow hotfix branch procedure (if defined)
2. Document bypass rationale in PR
3. Obtain approval from team lead or manager
4. Still require CI checks and code review (expedited)
5. Post-incident review of emergency change

## Enforcement

### Platform Configuration

Configure branch protection in your source control platform:

- **GitHub**: Settings → Branches → Branch protection rules
- **GitLab**: Settings → Repository → Protected branches
- **Bitbucket**: Repository settings → Branch permissions
- **Azure DevOps**: Repos → Branches → Branch policies

### Monitoring and Compliance

- Audit branch protection settings quarterly
- Review and update rules as team practices evolve
- Document any exceptions with business justification
- Include branch protection compliance in security reviews

## Benefits

**Quality**: All changes are tested and reviewed before reaching protected branches.

**Security**: Automated scans catch vulnerabilities before merge.

**Knowledge Sharing**: Code review ensures multiple team members understand changes.

**Auditability**: Pull request history provides clear record of what changed, why, and who approved it.

**Stability**: Protected branches remain stable and deployable at all times.

## Common Pitfalls

**Pitfall**: Granting bypass permissions too freely.

**Solution**: Limit bypass to genuine emergencies; document all uses.

**Pitfall**: Skipping CI checks to "save time."

**Solution**: If checks are too slow, optimize them - don't bypass them.

**Pitfall**: Rubber-stamp approvals without real review.

**Solution**: Foster code review culture; provide review guidelines and training.

**Pitfall**: Allowing stale branches to merge without updating.

**Solution**: Require branches to be current; automate merge conflict detection.

## Related Guidelines

- Code Review Practices
- Trunk-Based Development Strategy
- Continuous Integration Requirements
- Security Scanning and Compliance
