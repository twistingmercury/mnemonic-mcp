// Package routing implements the prompt routing engine for Mnemonic.
//
// It provides rule caching, priority-ordered evaluation, and pluggable matchers
// (keyword, regex, pattern/semantic) for routing prompts to agents.
//
// Documentation:
//   - Architecture: docs/architecture/03-system-architecture.md (Component Breakdown > Mnemonic)
//   - Architecture: docs/architecture/04-communication-patterns.md (CLI to Mnemonic Communication, Request Flow)
//   - Design: docs/design/routing-engine.md (Cache Architecture, Evaluator Interface, Evaluation Algorithm, Match Type Implementations, Confidence Scoring)
package routing
