package screenshot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/d0ughb0yy/goSpy/internal/models"
)

// Taker captures screenshots using a browser context.
type Taker struct {
	width  int
	height int
}

// NewTaker creates a Taker with the given viewport dimensions.
func NewTaker(width, height int) *Taker {
	return &Taker{
		width:  width,
		height: height,
	}
}

// Capture navigates to the target URL and captures a full-page screenshot.
// Returns the screenshot bytes. The target's ScreenshotPath is NOT set here —
// that's the output writer's job.
func (t *Taker) Capture(ctx context.Context, target *models.Target) ([]byte, error) {
	if target.Error != "" {
		return nil, fmt.Errorf(target.Error)
	}

	start := time.Now()

	// Set viewport and navigate
	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(int64(t.width), int64(t.height)),
		chromedp.Navigate(target.URL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.FullScreenshot(&buf, 90),
	); err != nil {
		return nil, fmt.Errorf("screenshot %s: %w", target.URL, err)
	}

	slog.Debug("screenshot captured",
		"url", target.URL,
		"size", len(buf),
		"duration", time.Since(start),
	)

	return buf, nil
}
