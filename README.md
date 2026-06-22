# goSpy

[ goSpy ] is a Go reconnaissance tool I built to capture web screenshots and metadata at scale.

I needed something that could quickly fetch a lot of URLs, detect what tech they're running, take full-page screenshots, and give me a clean report to flip through. Most existing tools were either too heavy (Selenium) or too limited. So I wrote this.

It runs a two-stage pipeline — first it HTTP-fetches everything with wappalyzer tech detection, then it spins up a headless browser to take screenshots of whatever succeeded. All of it runs through a configurable worker pool so you're not waiting forever on slow sites.

## Features

- **HTTP fetch** — records status codes, final URLs (follows redirects), response times, and skips bad certs
- **Tech detection** — uses wappalyzer to fingerprint frameworks and libraries from response headers + body
- **Screenshots** — full-page captures via chromedp with configurable viewport sizes
- **Profiles** — desktop (1920×1080), laptop (1366×768), or mobile (390×844)
- **Worker pool** — fan-out/fan-in concurrency with panic recovery and graceful Ctrl+C handling
- **HTML report** — self-contained with base64-embedded screenshots, tech tags, status code filtering, and redirect grouping
- **JSONL output** — one JSON object per target, easy to pipe into other tools

## Install

```bash
go install github.com/d0ughb0yy/goSpy/cmd/gospy@latest
```

You'll need a Chromium-based browser installed — google-chrome, chromium, brave, or similar. It auto-detects what you have.

## Usage

```bash
# Scan URLs from a file (one per line)
gospy scan targets.txt

# Single URL
gospy scan https://example.com

# Pipe URLs in from stdin
cat targets.txt | gospy scan -

# Custom output directory
gospy scan targets.txt -o ./results

# Mobile profile
gospy scan targets.txt --profile mobile

# Custom viewport and more workers
gospy scan targets.txt --width 1280 --height 720 --workers 8
```

## Output

```
output/
  screenshots/
    example.com.png
    ...
  results.jsonl
  report.html
```

`screenshots/*.png` are named by URL slug. `results.jsonl` has the raw structured data per target. `report.html` is the full visual report with everything embedded — you can open it offline or send it to someone as a single file.
