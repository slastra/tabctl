package utils

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"
)

// ReadStdin reads all content from stdin
func ReadStdin() (string, error) {
	return ReadStdinWithTimeout(0)
}

// ReadStdinWithTimeout reads from stdin with a timeout
func ReadStdinWithTimeout(timeout time.Duration) (string, error) {
	var input strings.Builder

	if timeout > 0 {
		// Set up timeout
		done := make(chan bool, 1)
		go func() {
			time.Sleep(timeout)
			done <- true
		}()

		// Read with timeout
		select {
		case <-done:
			return "", nil // Timeout reached, return empty
		default:
			// Continue with normal reading
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input.WriteString(scanner.Text())
		input.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.TrimSpace(input.String()), nil
}

// ReadStdinLines reads lines from stdin
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

// ReadStdinTabPairs reads tab ID and URL pairs from stdin
func ReadStdinTabPairs() (map[string]string, error) {
	pairs := make(map[string]string)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			tabID := strings.TrimSpace(parts[0])
			url := strings.TrimSpace(parts[1])
			pairs[tabID] = url
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return pairs, nil
}

// HasStdinData checks if there's data available on stdin
func HasStdinData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if stdin is a pipe or regular file (not a terminal)
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// WriteToStdout writes data to stdout
func WriteToStdout(data []byte) error {
	_, err := os.Stdout.Write(data)
	return err
}

// WriteLineToStdout writes a line to stdout
func WriteLineToStdout(line string) error {
	_, err := os.Stdout.WriteString(line + "\n")
	return err
}

// WriteToStderr writes data to stderr
func WriteToStderr(data []byte) error {
	_, err := os.Stderr.Write(data)
	return err
}

// ReadFromReader reads all data from a reader
func ReadFromReader(reader io.Reader) (string, error) {
	var content strings.Builder
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.TrimSpace(content.String()), nil
}

// IsTerminal checks if the file descriptor is a terminal
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