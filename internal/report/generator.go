package report

import (
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/d0ughb0yy/goSpy/internal/models"
)

//go:embed template.html
var templateFS embed.FS

type reportEntry struct {
	URL          string
	FinalURL     string
	StatusCode   int
	Technologies []string
	Error        string
	Duration     string
	ImageData    template.URL
}

type reportData struct {
	Title     string
	Generated string
	Total     int
	Success   int
	Failed    int
	Entries   []reportEntry
}

func Generate(outputDir string, targets []models.Target) error {
	total := len(targets)
	success := 0
	failed := 0

	entries := make([]reportEntry, 0, total)
	for _, t := range targets {
		if t.Error != "" {
			failed++
		} else {
			success++
		}

		entry := reportEntry{
			URL:          t.URL,
			FinalURL:     t.FinalURL,
			StatusCode:   t.StatusCode,
			Technologies: t.Technologies,
			Error:        t.Error,
			Duration:     formatDuration(t.Duration),
		}

		if t.ScreenshotPath != "" {
			data, err := os.ReadFile(t.ScreenshotPath)
			if err != nil {
				slog.Warn("report: read screenshot", "url", t.URL, "error", err)
			} else {
				mime := mimeType(t.ScreenshotPath)
				b64 := base64.StdEncoding.EncodeToString(data)
				entry.ImageData = template.URL(fmt.Sprintf("data:%s;base64,%s", mime, b64))
			}
		}

		entries = append(entries, entry)
	}

	tmpl, err := template.ParseFS(templateFS, "template.html")
	if err != nil {
		return fmt.Errorf("parse report template: %w", err)
	}

	data := reportData{
		Title:     "goSpy Scan Report",
		Generated: time.Now().Format("2006-01-02 15:04:05"),
		Total:     total,
		Success:   success,
		Failed:    failed,
		Entries:   entries,
	}

	path := filepath.Join(outputDir, "report.html")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("render report: %w", err)
	}

	slog.Debug("report generated", "path", path)
	return nil
}

func mimeType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%.2fm", d.Minutes())
}
