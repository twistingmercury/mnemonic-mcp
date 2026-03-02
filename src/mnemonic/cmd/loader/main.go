// Package main implements the pattern loader binary.
// It loads pattern files from a directory into Mnemonic via the Admin API.
//
// Usage:
//
//	loader --dir /path/to/patterns --api-url http://localhost:8080
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type frontmatter struct {
	EntityName      string   `yaml:"entity_name"`
	EntityType      string   `yaml:"entity_type"`
	Language        string   `yaml:"language"`
	Domain          string   `yaml:"domain"`
	Description     string   `yaml:"description"`
	Tags            []string `yaml:"tags"`
	Version         string   `yaml:"version"`
	RelatedPatterns []string `yaml:"related_patterns"`
}

type patternRequest struct {
	Name            string   `json:"name"`
	EntityType      string   `json:"entity_type"`
	Language        string   `json:"language"`
	Domain          string   `json:"domain"`
	Version         *string  `json:"version,omitempty"`
	Description     string   `json:"description,omitempty"`
	Content         string   `json:"content"`
	Tags            []string `json:"tags"`
	RelatedPatterns []string `json:"related_patterns"`
}

func main() {
	dir := flag.String("dir", "", "directory containing pattern .md files (required)")
	apiURL := flag.String("api-url", "http://localhost:8080", "Mnemonic Admin API base URL")
	flag.Parse()

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "error: --dir is required")
		os.Exit(1)
	}

	var failed int
	err := filepath.WalkDir(*dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		base := filepath.Base(path)
		if base == "README.md" || base == "PATTERN-METADATA-SCHEMA.md" {
			return nil
		}
		if loadErr := loadFile(path, *apiURL); loadErr != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", path, loadErr)
			failed++
		} else {
			fmt.Printf("OK   %s\n", path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "walk error: %v\n", err)
		os.Exit(1)
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "\n%d file(s) failed\n", failed)
		os.Exit(1)
	}
}

func loadFile(path, apiURL string) error {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}

	fm, content, err := parseFrontmatter(string(raw))
	if err != nil {
		return fmt.Errorf("parse frontmatter: %w", err)
	}

	name := slugFromFilename(filepath.Base(path))

	req := patternRequest{
		Name:            name,
		EntityType:      fm.EntityType,
		Language:        fm.Language,
		Domain:          fm.Domain,
		Description:     fm.Description,
		Content:         content,
		Tags:            fm.Tags,
		RelatedPatterns: fm.RelatedPatterns,
	}
	if fm.Version != "" {
		req.Version = &fm.Version
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.RelatedPatterns == nil {
		req.RelatedPatterns = []string{}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal pattern request: %w", err)
	}

	// Try POST first; fall back to PUT on 409 Conflict (pattern already exists).
	resp, err := http.Post(apiURL+"/v1/api/patterns", "application/json", bytes.NewReader(body)) //nolint:noctx
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusConflict {
		putReq, err := http.NewRequest(http.MethodPut,
			apiURL+"/v1/api/patterns/"+name, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build PUT request: %w", err)
		}
		putReq.Header.Set("Content-Type", "application/json")
		// G704: apiURL is an operator-supplied CLI flag, not user input — false positive.
		putResp, putErr := http.DefaultClient.Do(putReq) //#nosec G704 -- nolint:gosec
		if putErr != nil {
			return putErr
		}
		defer func() { _ = putResp.Body.Close() }()
		if putResp.StatusCode >= 300 {
			return fmt.Errorf("PUT returned %d", putResp.StatusCode)
		}
		return nil
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("POST returned %d", resp.StatusCode)
	}
	return nil
}

func parseFrontmatter(raw string) (*frontmatter, string, error) {
	parts := strings.SplitN(raw, "---", 3)
	if len(parts) < 3 {
		return nil, raw, fmt.Errorf("no YAML frontmatter found")
	}
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil, "", err
	}
	return &fm, strings.TrimSpace(parts[2]), nil
}

func slugFromFilename(name string) string {
	return strings.TrimSuffix(name, ".md")
}
