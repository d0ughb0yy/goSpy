package pipeline

import (
    "context"
    "fmt"
    "log/slog"

    "github.com/d0ughb0yy/goSpy/internal/browser"
    "github.com/d0ughb0yy/goSpy/internal/config"
    "github.com/d0ughb0yy/goSpy/internal/httpcollector"
    "github.com/d0ughb0yy/goSpy/internal/models"
    "github.com/d0ughb0yy/goSpy/internal/output"
    "github.com/d0ughb0yy/goSpy/internal/report"
    "github.com/d0ughb0yy/goSpy/internal/scheduler"
    "github.com/d0ughb0yy/goSpy/internal/screenshot"
)

// Run executes the full two-stage pipeline: HTTP fetch → screenshot.
func Run(ctx context.Context, cfg *config.Config, targets []models.Target) error {
	collector, err := httpcollector.New()
	if err != nil {
		return fmt.Errorf("init http collector: %w", err)
	}

	mgr, err := browser.NewManager(ctx)
	if err != nil {
		return fmt.Errorf("init browser: %w", err)
	}
	defer mgr.Close()

	taker := screenshot.NewTaker(cfg.Width, cfg.Height)
	writer := output.NewWriter(cfg.OutputDir)
	sched := scheduler.New(cfg.Workers)

	// Stage 1: HTTP fetch
	slog.Info("stage 1: fetching URLs", "count", len(targets))
	httpResults := sched.Run(ctx, targets, func(ctx context.Context, t *models.Target) {
		collector.Fetch(ctx, t)
	})

	httpSuccess, httpFailed := countResults(httpResults)
	slog.Info("stage 1 complete", "success", httpSuccess, "failed", httpFailed)

	// Stage 2: Screenshot
    var screenshotTargets []models.Target
    for _, t := range httpResults {
        if t.Error == "" {
            screenshotTargets = append(screenshotTargets, t)
        }
    }

	if len(screenshotTargets) > 0 {
		slog.Info("stage 2: capturing screenshots", "count", len(screenshotTargets))
        screenshotResults := sched.Run(ctx, screenshotTargets, func(ctx context.Context, t *models.Target) {
            tabCtx, tabCancel := mgr.NewContext()
            if tabCtx == nil {
                t.Error = "browser manager closed"
                return
            }
            defer tabCancel()

            // Bridge: cancel tab if pipeline context is cancelled (e.g., Ctrl+C)
            done := make(chan struct{})
            defer close(done)
            go func() {
                select {
                case <-ctx.Done():
                    tabCancel()
                case <-done:
                }
            }()

            // Apply timeout to screenshot context
            tabCtx, cancel := context.WithTimeout(tabCtx, cfg.Timeout)
            defer cancel()

            data, err := taker.Capture(tabCtx, t)
            if err != nil {
                t.Error = err.Error()
                return
            }

            path, err := writer.SaveScreenshot(data, t.URL)
            if err != nil {
                t.Error = err.Error()
                return
            }
            t.ScreenshotPath = path
        })

		screenshotMap := make(map[string]models.Target)
		for _, t := range screenshotResults {
			screenshotMap[t.URL] = t
		}
		for i := range httpResults {
			if updated, ok := screenshotMap[httpResults[i].URL]; ok {
				httpResults[i] = updated
			}
		}
	}

	slog.Info("saving results")
	if err := writer.SaveMetadata(httpResults); err != nil {
		return fmt.Errorf("save metadata: %w", err)
	}

	if err := report.Generate(cfg.OutputDir, httpResults); err != nil {
		slog.Warn("report generation failed", "error", err)
	}

	finalSuccess, finalFailed := countResults(httpResults)
	slog.Info("pipeline complete",
		"total", len(httpResults),
		"success", finalSuccess,
		"failed", finalFailed,
	)

	return nil
}

func countResults(targets []models.Target) (success, failed int) {
	for _, t := range targets {
		if t.Error != "" {
			failed++
		} else {
			success++
		}
	}
	return
}
