# goSpy — agent guide

## Build

```bash
go build ./...
```

No lint/typecheck scripts configured yet.

## Entrypoint

`cmd/gospy/main.go` — cobra root. Only subcommand: `scan` (`cmd/gospy/scan.go`).

## Architecture

Two-stage pipeline (`internal/pipeline/pipeline.go`):
1. **HTTP fetch** — `httpcollector` fetches, records status/FinalURL/tech, follows redirects
2. **Screenshot** — `browser` manager + `screenshot` taker via chromedp

Concurrency via `scheduler` — fan-out/fan-in worker pool, default 4 workers, min 1.

## Critical gotchas

- **Browser cleanup**: `browser/manager.go` generates a unique `gospy-{hex}` managerID per session. Cleanup in `Close()` scans `/proc/*/cmdline` for that ID + `headless`. **Never use `pkill -f headless`** — it kills user's main Brave (crashpad handlers contain "headless" in their path).
- **Target.Error is `string`**: stored as string, not `error` interface. `encoding/json` cannot serialize `error` properly for JSONL output.
- **Report generation is non-fatal**: `report.Generate()` failure logs `slog.Warn` and continues. Primary output is JSONL + PNGs.
- **Profile vs flags**: `scan.go` applies profile presets (desktop/laptop/mobile) only if user did not explicitly pass `--width`/`--height`. Checked via `cmd.Flags().Changed("width")`.
- **chromedp logging**: `chromedp.WithLogf`/`WithErrorf` return `BrowserOption`, not `ContextOption`. Can only be passed to the first `NewContext(allocCtx, ...)`. Passing them to tab-level `NewContext(m.browserCtx, ...)` panics. Child tabs inherit the suppression from the browser context.
- **No checkpoint/resume**: `--resume` flag exists but is a placeholder. Not implemented.

## Key dependencies

- `chromedp` — browser automation
- `wappalyzergo` — tech detection
- `cobra` — CLI

## Output

- `screenshots/*.png` — URL-slug filenames
- `results.jsonl` — JSON lines, one Target per line
- `report.html` — base64-embedded images, self-contained

## Input

Accepts file path, single URL argument, or stdin (`-`). File = one URL per line, comments/blanks ignored.

## Browser detection

Auto-detects in order: google-chrome, google-chrome-stable, chromium, chromium-browser, brave. Scans `$PATH` + common install paths.

## Testing

No unit tests yet. Verification done via `go build ./...` and manual e2e runs.
