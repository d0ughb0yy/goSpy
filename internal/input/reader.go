package input

import (
	"bufio"
	"os"
	"strings"

	"github.com/d0ughb0yy/goSpy/internal/models"
)

// ReadTargets reads URLs from a file (one per line) and returns Target objects.
// If path is empty, reads from stdin.
func ReadTargets(path string) ([]models.Target, error) {
	var scanner *bufio.Scanner

	if path == "" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		scanner = bufio.NewScanner(f)
	}

	var targets []models.Target
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "http://") && !strings.HasPrefix(line, "https://") {
			line = "https://" + line
		}
		targets = append(targets, models.Target{URL: line})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}

// SingleTarget creates a Target from a single URL argument.
func SingleTarget(url string) models.Target {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	return models.Target{URL: url}
}
