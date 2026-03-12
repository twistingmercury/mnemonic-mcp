# Enriched Chunk Embedding Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prepend pattern name, tags, and section title to each chunk's text before generating its embedding, so semantic search retrieves relevant patterns even for natural-language queries.

**Architecture:** In `processChunkJob`, after loading the chunk, load its parent pattern to retrieve `Name` and `Tags`. Build an enriched text string (`"{name} | {tags} | {section_title}\n\n{content}"`) and pass that to `embeddingSvc.Embed` instead of raw `chunk.Content`. All error handling follows the existing `failChunkJob` convention.

**Tech Stack:** Go, `github.com/twistingmercury/mnemonic` internal packages, testify mocks

---

## Files

- Modify: `src/mnemonic/internal/service/enrichment/service.go` — `processChunkJob` function (~line 134)
- Modify: `src/mnemonic/internal/service/enrichment/service_test.go` — update chunk job tests, add new failure test

---

## Chunk 1: Implementation + Tests

### Task 1: Add failing test for pattern-load failure in chunk pipeline

**Files:**
- Modify: `src/mnemonic/internal/service/enrichment/service_test.go`

Context: `testChunk()` currently returns a `Chunk` with no `SectionTitle`. The enriched text format is:
```
{pattern.Name} | {tags joined by ", "} | {chunk.SectionTitle}\n\n{chunk.Content}
```

For `testPattern()`: name=`"test-pattern"`, tags=`["test"]`, for `testChunk()` with `SectionTitle="Test Section"`, the enriched text will be:
```
test-pattern | test | Test Section\n\nTest chunk content for embedding.
```

- [ ] **Step 1: Add `SectionTitle` to `testChunk()`**

In `service_test.go`, find `func testChunk()` (~line 444) and add `SectionTitle`:

```go
func testChunk() *chunkrepo.Chunk {
	return &chunkrepo.Chunk{
		ID:           testChunkID,
		PatternID:    testPatternID,
		SectionTitle: "Test Section",
		Content:      "Test chunk content for embedding.",
	}
}
```

- [ ] **Step 2: Add helper for enriched embed text**

Add this helper near `testChunk()`:

```go
func testEnrichedEmbedText() string {
	chunk := testChunk()
	pattern := testPattern()
	tags := strings.Join(pattern.Tags, ", ")
	return fmt.Sprintf("%s | %s | %s\n\n%s", pattern.Name, tags, chunk.SectionTitle, chunk.Content)
}
```

Add `"fmt"` and `"strings"` to imports if not already present.

- [ ] **Step 3: Write the failing test for pattern-load failure**

Add after `TestProcessJob_ChunkJob_Step1_LoadChunkFails`:

```go
func TestProcessJob_ChunkJob_Step2_LoadPatternFails(t *testing.T) {
	t.Parallel()

	svc, deps := newTestService(t)
	chunk := testChunk()

	deps.chunkRepo.On("Get", mock.Anything, testChunkID).Return(chunk, nil)
	deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(nil, errors.New("db error"))

	// failChunkJob: update chunk status, update pattern status, mark job failed.
	deps.chunkRepo.On("UpdateEnrichmentStatus", mock.Anything, testChunkID, "failed", mock.AnythingOfType("*string")).Return(nil)
	deps.patternRepo.On("UpdateEnrichmentStatus", mock.Anything, testPatternID, "failed", mock.AnythingOfType("*string")).Return(nil)
	deps.jobRepo.On("MarkFailed", mock.Anything, testJobID, mock.Anything, 30*time.Second).Return(nil)

	err := svc.ProcessJob(context.Background(), testChunkJob())

	require.NoError(t, err, "pipeline failure should return nil when failChunkJob succeeds")
	assertExpectations(t, deps)
}
```

- [ ] **Step 4: Run the new test to confirm it fails**

```bash
cd src/mnemonic && go test ./internal/service/enrichment/... -run TestProcessJob_ChunkJob_Step2_LoadPatternFails -v
```

Expected: FAIL (pattern load not yet implemented in processChunkJob)

---

### Task 2: Update existing chunk job tests to expect enriched embed text

The following tests mock `Embed` with `chunk.Content` and must be updated to use `testEnrichedEmbedText()`. Each also needs a `patternRepo.Get` mock added **before** the `Embed` call.

**Files:**
- Modify: `src/mnemonic/internal/service/enrichment/service_test.go`

- [ ] **Step 1: Update `TestProcessJob_ChunkJob_HappyPath_AllEnriched`**

Find (~line 934) and replace:
```go
// before
deps.embeddingSvc.On("Embed", mock.Anything, chunk.Content).Return(testEmbedding(), nil)
```
with:
```go
// after — pattern load happens before embed
deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(testPattern(), nil)
deps.embeddingSvc.On("Embed", mock.Anything, testEnrichedEmbedText()).Return(testEmbedding(), nil)
```

Note: `setupChunkGraphMocks` also sets up `deps.patternRepo.On("Get", ...)` for the graph pipeline step. After your change, `patternRepo.Get` will be called **twice** — once in `processChunkJob` and once in `runGraphPipeline`. Testify's mock records calls in order; `On("Get", ...)` set up twice (once here, once inside `setupChunkGraphMocks`) is fine — testify matches each call in sequence.

- [ ] **Step 2: Update `TestProcessJob_ChunkJob_HappyPath_NotAllEnriched`**

Find (~line 958) and replace:
```go
deps.embeddingSvc.On("Embed", mock.Anything, chunk.Content).Return(testEmbedding(), nil)
```
with:
```go
deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(testPattern(), nil)
deps.embeddingSvc.On("Embed", mock.Anything, testEnrichedEmbedText()).Return(testEmbedding(), nil)
```

- [ ] **Step 3: Update `TestProcessJob_ChunkJob_Step2_EmbeddingFails`**

This test is currently named "Step2" but will become "Step3" after the pattern-load step is inserted. Update the mock and add pattern load:

```go
deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(testPattern(), nil)
deps.embeddingSvc.On("Embed", mock.Anything, testEnrichedEmbedText()).Return(nil, errors.New("openai unavailable"))
```

- [ ] **Step 4: Update `TestProcessJob_ChunkJob_Step3_UpdateEmbeddingFails`**

Add pattern load mock and update embed expectation:
```go
deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(testPattern(), nil)
deps.embeddingSvc.On("Embed", mock.Anything, testEnrichedEmbedText()).Return(testEmbedding(), nil)
```

- [ ] **Step 5: Update `TestProcessJob_ChunkJob_Step4_UpdateChunkStatusFails`**

Add pattern load mock and update embed expectation:
```go
deps.patternRepo.On("Get", mock.Anything, testPatternID).Return(testPattern(), nil)
deps.embeddingSvc.On("Embed", mock.Anything, testEnrichedEmbedText()).Return(testEmbedding(), nil)
```

- [ ] **Step 6: Run all chunk job tests to confirm they all fail for the right reason**

```bash
cd src/mnemonic && go test ./internal/service/enrichment/... -run TestProcessJob_ChunkJob -v 2>&1 | grep -E "PASS|FAIL|---"
```

Expected: All FAIL (implementation not updated yet). The new `Step2_LoadPatternFails` test should also fail.

---

### Task 3: Implement enriched embedding in `processChunkJob`

**Files:**
- Modify: `src/mnemonic/internal/service/enrichment/service.go`

- [ ] **Step 1: Load parent pattern after loading chunk**

In `processChunkJob` (~line 134), after the chunk load block and before the embed call, add:

```go
// Step 2: Load parent pattern to build enriched embed text.
pattern, err := s.patternRepo.Get(ctx, chunk.PatternID)
if err != nil {
    return s.failChunkJob(ctx, job, chunk.ID, chunk.PatternID, fmt.Errorf("load pattern for chunk: %w", err))
}
```

- [ ] **Step 2: Build enriched text and pass it to Embed**

Replace:
```go
// Step 3: Generate embedding for chunk content.
embedding, err := s.embeddingSvc.Embed(ctx, chunk.Content)
```

with:
```go
// Step 3: Generate embedding for enriched chunk text.
// Prepend pattern name, tags, and section title so the embedding captures
// semantic context beyond the raw code/prose of the section body.
tags := strings.Join(pattern.Tags, ", ")
embedText := fmt.Sprintf("%s | %s | %s\n\n%s", pattern.Name, tags, chunk.SectionTitle, chunk.Content)
embedding, err := s.embeddingSvc.Embed(ctx, embedText)
```

Add `"strings"` to the import block if not already present.

- [ ] **Step 3: Run the full enrichment test suite**

```bash
cd src/mnemonic && go test ./internal/service/enrichment/... -v 2>&1 | grep -E "PASS|FAIL|---"
```

Expected: All tests PASS.

- [ ] **Step 4: Run the full service test suite to check for regressions**

```bash
cd src/mnemonic && go test ./... 2>&1 | tail -30
```

Expected: All packages pass, exit 0.

- [ ] **Step 5: Commit**

```bash
cd src/mnemonic && git add internal/service/enrichment/service.go internal/service/enrichment/service_test.go
git commit -m "feat: prepend pattern metadata to chunk embed text for better semantic search"
```
