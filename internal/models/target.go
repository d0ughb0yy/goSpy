package models

import "time"

// Target represents a URL being processed through the pipeline.
type Target struct {
	URL            string        `json:"url"`
	FinalURL       string        `json:"final_url,omitempty"`
	StatusCode     int           `json:"status_code,omitempty"`
	Title          string        `json:"title,omitempty"`
	Technologies   []string      `json:"technologies,omitempty"`
	ScreenshotPath string        `json:"screenshot_path,omitempty"`
	Error          string        `json:"error,omitempty"`
	Duration       time.Duration `json:"duration,omitempty"`
}
