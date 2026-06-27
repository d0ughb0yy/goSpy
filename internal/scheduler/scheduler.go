package scheduler

import (
    "context"
    "fmt"
    "log/slog"
    "math/rand"
    "sync"
    "time"

    "github.com/d0ughb0yy/goSpy/internal/models"
)

// ProcessFunc is a function that processes a target in place.
type ProcessFunc func(ctx context.Context, target *models.Target)

// Scheduler distributes targets across a worker pool.
type Scheduler struct {
	workers int
	delay   time.Duration
}

// New creates a Scheduler with the given worker count.
// If workers <= 0, defaults to 1.
func New(workers int) *Scheduler {
    if workers <= 0 {
        workers = 1
    }
    return &Scheduler{workers: workers}
}

// WithDelay returns a copy of the Scheduler with a per-request delay applied between each target.
func (s *Scheduler) WithDelay(d time.Duration) *Scheduler {
    s.delay = d
    return s
}

// Run processes all targets through the given function using a worker pool.
// It blocks until all targets are processed or the context is cancelled.
func (s *Scheduler) Run(ctx context.Context, targets []models.Target, fn ProcessFunc) []models.Target {
	if len(targets) == 0 {
		return nil
	}

	input := make(chan models.Target, len(targets))
	output := make(chan models.Target, len(targets))

	// Feed input channel
	for _, t := range targets {
		input <- t
	}
	close(input)

	// Start workers
	var wg sync.WaitGroup
    for i := 0; i < s.workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for target := range input {
                func() {
                    defer func() {
                        if r := recover(); r != nil {
                            target.Error = fmt.Sprintf("panic: %v", r)
                            output <- target
                        }
                    }()

                    select {
                    case <-ctx.Done():
                        target.Error = ctx.Err().Error()
                        output <- target
                        return
                    default:
                    }

                    fn(ctx, &target)
                    output <- target

                    // Apply jittered delay between requests (up to +500ms)
                    if s.delay > 0 {
                        jitter := time.Duration(rand.Int63n(500)) * time.Millisecond
                        select {
                        case <-ctx.Done():
                            return
                        case <-time.After(s.delay + jitter):
                        }
                    }
                }()
            }
        }()
    }

	// Close output channel when all workers done
	go func() {
		wg.Wait()
		close(output)
	}()

	// Collect results
	var results []models.Target
	for target := range output {
		results = append(results, target)
	}

	slog.Debug("scheduler complete",
		"input", len(targets),
		"output", len(results),
	)

	return results
}
