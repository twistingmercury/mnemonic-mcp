// White-box test file: accesses unexported splitIntoChunks directly.
package pattern

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitChunks_ByH2Heading(t *testing.T) {
	content := "## Philosophy\nStorage-only databases.\n\n## created_at Handling\nLet the database set it.\n\n## updated_at Handling\nAlways set explicitly."
	chunks := splitIntoChunks(content)
	require.Len(t, chunks, 3)
	assert.Equal(t, "Philosophy", chunks[0].SectionTitle)
	assert.Equal(t, 0, chunks[0].ChunkIndex)
	assert.Contains(t, chunks[0].Content, "Storage-only")
	assert.Equal(t, "updated_at Handling", chunks[2].SectionTitle)
	assert.Equal(t, 2, chunks[2].ChunkIndex)
}

func TestSplitChunks_NoH2_SingleChunk(t *testing.T) {
	content := "Just a paragraph with no headings."
	chunks := splitIntoChunks(content)
	require.Len(t, chunks, 1)
	assert.Equal(t, "Content", chunks[0].SectionTitle)
	assert.Equal(t, content, chunks[0].Content)
}

func TestSplitChunks_ContentBeforeFirstH2(t *testing.T) {
	content := "Preamble text here.\n\n## First Section\nSection content."
	chunks := splitIntoChunks(content)
	require.Len(t, chunks, 2)
	assert.Equal(t, "Overview", chunks[0].SectionTitle)
	assert.Contains(t, chunks[0].Content, "Preamble")
	assert.Equal(t, "First Section", chunks[1].SectionTitle)
}

func TestSplitChunks_EmptySectionDropped(t *testing.T) {
	content := "## Empty\n\n## Real Section\nActual content here."
	chunks := splitIntoChunks(content)
	require.Len(t, chunks, 1)
	assert.Equal(t, "Real Section", chunks[0].SectionTitle)
	assert.Equal(t, 0, chunks[0].ChunkIndex)
}

func TestSplitChunks_EmptyString(t *testing.T) {
	chunks := splitIntoChunks("")
	require.Len(t, chunks, 0)
}

func TestSplitChunks_SingleH2WithContent(t *testing.T) {
	content := "## Only Section\nSome content here."
	chunks := splitIntoChunks(content)
	require.Len(t, chunks, 1)
	assert.Equal(t, "Only Section", chunks[0].SectionTitle)
	assert.Equal(t, 0, chunks[0].ChunkIndex)
	assert.Equal(t, "Some content here.", chunks[0].Content)
}

func TestSplitChunks_H2TrailingWhitespace(t *testing.T) {
	content := "## Section With Spaces   \nContent here."
	chunks := splitIntoChunks(content)
	require.Len(t, chunks, 1)
	// TrimPrefix leaves trailing whitespace — section title should match raw suffix.
	assert.Equal(t, "Section With Spaces   ", chunks[0].SectionTitle)
	assert.Equal(t, "Content here.", chunks[0].Content)
}

func TestSplitChunks_WhitespaceOnlyContent(t *testing.T) {
	chunks := splitIntoChunks("   \n\t\n  ")
	require.Len(t, chunks, 0)
}
