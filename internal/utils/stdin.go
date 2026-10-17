package utils

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// maxLineSize bounds one stdin line. bufio.Scanner's 64KB default rejected
// long data: URLs piped to `tabctl open`. 1MB matches the largest message a
// native-messaging host may send the browser, so anything longer could not
// reach the extension anyway.
const maxLineSize = 1 << 20

// ReadStdinLines reads stdin line-by-line, trimming whitespace and skipping
// blank lines.
func ReadStdinLines() ([]string, error) {
	return readLines(os.Stdin)
}

func readLines(r io.Reader) ([]string, error) {
	var lines []string

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
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
