package utils

import (
	"bufio"
	"os"
	"strings"
)

// ReadStdinLines reads stdin line-by-line, trimming whitespace and skipping
// blank lines.
func ReadStdinLines() ([]string, error) {
	var lines []string

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// IsTerminal reports whether the given fd refers to a terminal. Only the
// standard fds (stdin/stdout/stderr) are recognized — anything else returns
// false.
func IsTerminal(fd int) bool {
	switch fd {
	case int(os.Stdin.Fd()):
		stat, err := os.Stdin.Stat()
		return err == nil && (stat.Mode()&os.ModeCharDevice) != 0
	case int(os.Stdout.Fd()):
		stat, err := os.Stdout.Stat()
		return err == nil && (stat.Mode()&os.ModeCharDevice) != 0
	case int(os.Stderr.Fd()):
		stat, err := os.Stderr.Stat()
		return err == nil && (stat.Mode()&os.ModeCharDevice) != 0
	default:
		return false
	}
}
