package httpcollector

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	wappalyzer "github.com/projectdiscovery/wappalyzergo"
	"github.com/d0ughb0yy/goSpy/internal/config"
	"github.com/d0ughb0yy/goSpy/internal/models"
)

// Collector performs HTTP fetches and technology detection.
type Collector struct {
	client      *http.Client
	wapp        *wappalyzer.Wappalyze
	maxPerHost  int
	maxRetries  int
	noRateLimit bool
	hostSem     map[string]chan struct{}
	hostMu      sync.Mutex
}

// New creates a Collector with a default HTTP client and wappalyzer.
func New(cfg *config.Config) (*Collector, error) {
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
		wapp:        w,
		maxPerHost:  cfg.MaxPerHost,
		maxRetries:  cfg.MaxRetries,
		noRateLimit: cfg.NoRateLimit,
		hostSem:     make(map[string]chan struct{}),
	}, nil
}

// semaphoreForHost returns a per-host semaphore channel, creating one if needed.
func (c *Collector) semaphoreForHost(host string) chan struct{} {
	c.hostMu.Lock()
	defer c.hostMu.Unlock()
	sem, ok := c.hostSem[host]
	if !ok {
		sem = make(chan struct{}, c.maxPerHost)
		c.hostSem[host] = sem
	}
	return sem
}

// Fetch performs an HTTP GET on the target URL, records status code,
// detects technologies, and updates the Target in place.
func (c *Collector) Fetch(ctx context.Context, target *models.Target) {
	start := time.Now()

	// Parse host for per-host limiting
	u, err := url.Parse(target.URL)
	if err != nil {
		target.Error = fmt.Sprintf("parse url: %s", err)
		return
	}

	// Per-host concurrency limit
	if !c.noRateLimit && c.maxPerHost > 0 {
		sem := c.semaphoreForHost(u.Host)
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			target.Error = ctx.Err().Error()
			return
		}
		defer func() { <-sem }()
	}

	// Retry loop with exponential backoff for rate-limit responses
	maxAttempts := 1
	if !c.noRateLimit && c.maxRetries > 0 {
		maxAttempts = c.maxRetries + 1
	}

	var resp *http.Response
	var body []byte
	var doErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			jitter := time.Duration(rand.Int63n(1000)) * time.Millisecond
			select {
			case <-ctx.Done():
				target.Error = ctx.Err().Error()
				return
			case <-time.After(backoff + jitter):
			}
		}

		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
		if reqErr != nil {
			target.Error = fmt.Sprintf("create request: %s", reqErr)
			return
		}

		if c.noRateLimit {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		} else {
			req.Header.Set("User-Agent", randomUserAgent())
		}

		resp, doErr = c.client.Do(req)
		if doErr != nil {
			// Network error, not worth retrying if context cancelled
			target.Error = fmt.Sprintf("http get: %s", doErr)
			return
		}

		// Check for rate-limit status codes
		if !c.noRateLimit && (resp.StatusCode == 429 || resp.StatusCode == 503) {
			slog.Warn("rate limited, retrying",
				"url", target.URL,
				"status", resp.StatusCode,
				"attempt", attempt+1,
			)
			retryAfter := resp.Header.Get("Retry-After")
			resp.Body.Close()

			if retryAfter != "" && attempt < maxAttempts-1 {
				if d, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil {
					select {
					case <-ctx.Done():
						target.Error = ctx.Err().Error()
						return
					case <-time.After(d):
					}
				}
			}
			continue
		}
		defer resp.Body.Close()

		body, doErr = io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		resp.Body.Close()
		if doErr != nil {
			target.Error = fmt.Sprintf("read body: %s", doErr)
			return
		}
		break
	}

	if doErr != nil {
		return
	}
	if resp == nil {
		target.Error = "all retries exhausted"
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
