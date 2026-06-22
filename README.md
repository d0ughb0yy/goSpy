# goSpy

A reconnaissance platform for capturing web screenshots and metadata at scale.

## Features

- **HTTP Collection** — fetches each target, records status code, final URL, and response time
- **Technology Detection** — identifies frameworks and libraries via wappalyzer
- **Screenshots** — full-page screenshots using headless Chromium/Chrome/Brave via chromedp
- **Profiles** — desktop (1920×1080), laptop (1366×768), mobile (390×844)
- **Concurrency** — configurable worker pool with panic recovery and graceful shutdown
- **HTML Report** — self-contained report with base64-embedded screenshots, tech tags, and status code filtering
- **JSONL Output** — structured metadata per target for further processing

## Install

```bash
go install github.com/d0ughb0yy/goSpy/cmd/gospy@latest
```

Requires a Chromium-based browser (google-chrome, google-chrome-stable, chromium, chromium-browser, or brave).

## Usage

```bash
# Scan URLs from a file
gospy scan targets.txt

# Scan a single URL
gospy scan https://example.com

# Scan from stdin
cat targets.txt | gospy scan -

# Custom output directory
gospy scan targets.txt -o ./results

# Use a profile preset
gospy scan targets.txt --profile mobile

# Custom dimensions and worker count
gospy scan targets.txt --width 1280 --height 720 --workers 8
```

## Output

- `screenshots/*.png` — captured screenshots
- `results.jsonl` — structured metadata (URL, status, tech, error)
- `report.html` — self-contained visual report
