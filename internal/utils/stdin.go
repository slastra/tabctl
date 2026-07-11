package utils

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// ReadStdinLines reads stdin line-by-line, trimming whitespace and skipping
// blank lines.
func ReadStdinLines() ([]string, error) {
	return readLines(os.Stdin)
}

func readLines(r io.Reader) ([]string, error) {
	var lines []string

	scanner := bufio.NewScanner(r)
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
