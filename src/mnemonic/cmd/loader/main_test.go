package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalPatternMD is a valid .md file with frontmatter for testing.
const minimalPatternMD = `---
entity_name: go-error-handling
entity_type: pattern
language: go
domain: backend
description: How to handle errors in Go
tags:
  - errors
  - go
version: "1.0"
---

## Go Error Handling

Wrap errors with context using %%w.
`

func TestParseFrontmatter_ValidInput(t *testing.T) {
	t.Parallel()
	fm, content, err := parseFrontmatter(minimalPatternMD)
	require.NoError(t, err)
	assert.Equal(t, "pattern", fm.EntityType)
	assert.Equal(t, "go", fm.Language)
	assert.Equal(t, "backend", fm.Domain)
	assert.Equal(t, "How to handle errors in Go", fm.Description)
	assert.Equal(t, "1.0", fm.Version)
	assert.Equal(t, []string{"errors", "go"}, fm.Tags)
	assert.Contains(t, content, "Go Error Handling")
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	t.Parallel()
	_, _, err := parseFrontmatter("just plain text with no delimiters")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no YAML frontmatter found")
}

func TestSlugFromFilename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"go-error-handling.md", "go-error-handling"},
		{"pattern.md", "pattern"},
		{"no-extension", "no-extension"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, slugFromFilename(tc.input))
		})
	}
}

func TestLookupPatternID_Found(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/api/patterns", r.URL.Path)
		assert.Equal(t, "go-error-handling", r.URL.Query().Get("search"))

		resp := patternListEnvelope{
			Data: []patternListItem{
				{ID: "11111111-1111-1111-1111-111111111111", Name: "go-error-wrapping"},
				{ID: "22222222-2222-2222-2222-222222222222", Name: "go-error-handling"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	id, err := lookupPatternID(srv.URL, "go-error-handling")
	require.NoError(t, err)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", id)
}

func TestLookupPatternID_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := patternListEnvelope{
			Data: []patternListItem{
				{ID: "11111111-1111-1111-1111-111111111111", Name: "other-pattern"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	_, err := lookupPatternID(srv.URL, "go-error-handling")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "go-error-handling")
	assert.Contains(t, err.Error(), "not found")
}

func TestLookupPatternID_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := lookupPatternID(srv.URL, "go-error-handling")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestLoadFile_PostConflictFallback exercises the full 409 → GET → PUT upsert path.
func TestLoadFile_PostConflictFallback(t *testing.T) {
	t.Parallel()

	const patternUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	var putCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/api/patterns":
			// Simulate conflict on first create.
			w.WriteHeader(http.StatusConflict)

		case r.Method == http.MethodGet && r.URL.Path == "/v1/api/patterns":
			// Return list with the existing pattern's UUID.
			resp := patternListEnvelope{
				Data: []patternListItem{
					{ID: patternUUID, Name: "go-error-handling"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPut && r.URL.Path == "/v1/api/patterns/"+patternUUID:
			putCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	// Write a temp .md file.
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "go-error-handling.md"))
	require.NoError(t, err)
	_, err = f.WriteString(minimalPatternMD)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	err = loadFile(filepath.Join(dir, "go-error-handling.md"), srv.URL)
	require.NoError(t, err)
	assert.True(t, putCalled, "PUT to UUID endpoint must have been called")
}

// TestLoadFile_PostConflictFallback_LookupFails ensures lookupPatternID errors propagate.
func TestLoadFile_PostConflictFallback_LookupFails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusConflict)
		case r.Method == http.MethodGet:
			// Return empty list — pattern UUID can't be found.
			resp := patternListEnvelope{Data: []patternListItem{}}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "go-error-handling.md"))
	require.NoError(t, err)
	_, err = f.WriteString(minimalPatternMD)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	err = loadFile(filepath.Join(dir, "go-error-handling.md"), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lookup pattern UUID")
}
