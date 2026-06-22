package httpcollector

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	wappalyzer "github.com/projectdiscovery/wappalyzergo"
	"github.com/d0ughb0yy/goSpy/internal/models"
)

// Collector performs HTTP fetches and technology detection.
type Collector struct {
	client    *http.Client
	wapp *wappalyzer.Wappalyze
}

// New creates a Collector with a default HTTP client and wappalyzer.
func New() (*Collector, error) {
	w, err := wappalyzer.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize wappalyzer: %w", err)
	}

	return &Collector{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		wapp: w,
	}, nil
}

// Fetch performs an HTTP GET on the target URL, records status code,
// detects technologies, and updates the Target in place.
func (c *Collector) Fetch(ctx context.Context, target *models.Target) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		target.Error = fmt.Sprintf("create request: %s", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		target.Error = fmt.Sprintf("http get: %s", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		target.Error = fmt.Sprintf("read body: %s", err)
		return
	}

	target.FinalURL = resp.Request.URL.String()
	target.StatusCode = resp.StatusCode
	target.Duration = time.Since(start)

	// Detect technologies
	fingerprints := c.wapp.Fingerprint(resp.Header, body)
	techs := make([]string, 0, len(fingerprints))
	for name := range fingerprints {
		techs = append(techs, name)
	}
	target.Technologies = techs

	slog.Debug("fetched",
		"url", target.URL,
		"status", target.StatusCode,
		"technologies", len(techs),
		"duration", target.Duration,
	)
}
