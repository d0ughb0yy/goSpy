package output

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/d0ughb0yy/goSpy/internal/models"
)

const maxSlugLength = 200

var invalidChars = regexp.MustCompile(`[^a-zA-Z0-9\-_]`)

// slugify converts a URL to a filesystem-safe slug.
// Example: "https://example.com/admin" → "https---example-com-admin"
func slugify(rawURL string) string {
	decoded, err := url.QueryUnescape(rawURL)
	if err != nil {
		decoded = rawURL
	}
	s := strings.ReplaceAll(decoded, "://", "---")
	s = invalidChars.ReplaceAllString(s, "-")
	s = strings.TrimRight(s, "-")
	if len(s) > maxSlugLength {
		s = s[:maxSlugLength]
	}
	return s
}

// Writer saves screenshots and metadata to disk.
type Writer struct {
	outputDir string
}

// NewWriter creates a Writer that outputs to the given directory.
func NewWriter(outputDir string) *Writer {
	return &Writer{outputDir: outputDir}
}

// SaveScreenshot writes screenshot bytes to disk and returns the file path.
func (w *Writer) SaveScreenshot(data []byte, url string) (string, error) {
	dir := filepath.Join(w.outputDir, "screenshots")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create screenshots dir: %w", err)
	}

	filename := slugify(url) + ".png"
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write screenshot: %w", err)
	}

	return path, nil
}

// SaveMetadata writes all target metadata as JSONL.
func (w *Writer) SaveMetadata(targets []models.Target) error {
	if err := os.MkdirAll(w.outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	path := filepath.Join(w.outputDir, "results.jsonl")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create results file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, t := range targets {
		if err := enc.Encode(t); err != nil {
			return fmt.Errorf("encode target: %w", err)
		}
	}

	return nil
}
