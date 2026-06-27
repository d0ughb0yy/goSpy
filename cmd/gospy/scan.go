package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "time"

    "github.com/spf13/cobra"

    "github.com/d0ughb0yy/goSpy/internal/config"
    "github.com/d0ughb0yy/goSpy/internal/input"
    "github.com/d0ughb0yy/goSpy/internal/logger"
    "github.com/d0ughb0yy/goSpy/internal/models"
    "github.com/d0ughb0yy/goSpy/internal/pipeline"
)

var scanCmd = &cobra.Command{
	Use:   "scan [file-or-url]",
	Short: "Scan URLs for screenshots and technology detection",
	Long: `Scan targets from a file (one URL per line) or a single URL argument.
Output is saved to the specified directory.`,
	Args: cobra.ExactArgs(1),
	RunE: runScan,
}

var (
	workers     int
	width       int
	height      int
	timeout     time.Duration
	output      string
	profile     string
	resume      bool
	verbose     bool
	delay       time.Duration
	maxPerHost  int
	maxRetries  int
	noRateLimit bool
)

func init() {
	scanCmd.Flags().IntVarP(&workers, "workers", "w", 4, "number of concurrent workers")
	scanCmd.Flags().IntVar(&width, "width", 1920, "viewport width")
	scanCmd.Flags().IntVar(&height, "height", 1080, "viewport height")
	scanCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "per-target timeout")
	scanCmd.Flags().StringVarP(&output, "output", "o", "output", "output directory")
	scanCmd.Flags().StringVarP(&profile, "profile", "p", "desktop", "screenshot profile (desktop, laptop, mobile)")
	scanCmd.Flags().BoolVar(&resume, "resume", false, "resume from checkpoint (not yet implemented)")
	scanCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	scanCmd.Flags().DurationVar(&delay, "delay", 2*time.Second, "delay between requests (random jitter up to +500ms)")
	scanCmd.Flags().IntVar(&maxPerHost, "max-per-host", 2, "max concurrent requests per host")
	scanCmd.Flags().IntVar(&maxRetries, "max-retries", 3, "max retries on rate-limit responses (429/503)")
	scanCmd.Flags().BoolVar(&noRateLimit, "no-rate-limit", false, "disable all rate limiting")

	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
    logger.Setup(verbose)

    cfg := &config.Config{
        Workers:     workers,
        Width:       width,
        Height:      height,
        Timeout:     timeout,
        OutputDir:   output,
        Profile:     profile,
        Resume:      resume,
        Verbose:     verbose,
        Delay:       delay,
        MaxPerHost:  maxPerHost,
        MaxRetries:  maxRetries,
        NoRateLimit: noRateLimit,
    }

    // Apply profile presets only if user didn't explicitly set width/height
    widthChanged := cmd.Flags().Changed("width")
    heightChanged := cmd.Flags().Changed("height")
    if !widthChanged && !heightChanged {
        switch profile {
        case "mobile":
            cfg.Width = 390
            cfg.Height = 844
        case "laptop":
            cfg.Width = 1366
            cfg.Height = 768
        case "desktop":
            cfg.Width = 1920
            cfg.Height = 1080
        default:
            return fmt.Errorf("unknown profile %q (valid: desktop, laptop, mobile)", profile)
        }
    }

    // Read targets
    var targets []models.Target
    arg := args[0]

    if arg == "-" {
        // Read from stdin
        var readErr error
        targets, readErr = input.ReadTargets("")
        if readErr != nil {
            return fmt.Errorf("read targets from stdin: %w", readErr)
        }
    } else if _, statErr := os.Stat(arg); statErr == nil {
        // It's a file
        var readErr error
        targets, readErr = input.ReadTargets(arg)
        if readErr != nil {
            return fmt.Errorf("read targets: %w", readErr)
        }
    } else {
        // It's a URL
        targets = []models.Target{input.SingleTarget(arg)}
    }

    if len(targets) == 0 {
        return fmt.Errorf("no targets found")
    }

    // Set up context with signal handling
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    slog.Info("scanning", "targets", len(targets), "workers", cfg.Workers)

    return pipeline.Run(ctx, cfg, targets)
}
