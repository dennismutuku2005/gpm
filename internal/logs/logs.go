package logs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	MaxLogSize  = 10 * 1024 * 1024 // 10MB
	MaxLogFiles = 5                // Keep 5 backup logs
)

// RotateWriter is an io.WriteCloser that writes to a file and rotates it when it exceeds MaxLogSize.
type RotateWriter struct {
	mu       sync.Mutex
	filename string
	file     *os.File
	size     int64
}

// NewRotateWriter creates a new RotateWriter.
func NewRotateWriter(filename string) (*RotateWriter, error) {
	rw := &RotateWriter{filename: filename}
	if err := rw.open(); err != nil {
		return nil, err
	}
	return rw, nil
}

func (rw *RotateWriter) open() error {
	dir := filepath.Dir(rw.filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log dir: %w", err)
	}

	info, err := os.Stat(rw.filename)
	if err == nil {
		rw.size = info.Size()
	} else if !os.IsNotExist(err) {
		return err
	}

	file, err := os.OpenFile(rw.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	rw.file = file
	return nil
}

// Write writes to the log file, performing rotation if size limit is reached.
func (rw *RotateWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	writeSize := int64(len(p))
	if rw.size+writeSize > MaxLogSize {
		if err := rw.rotate(); err != nil {
			// Write anyway or return error
			return 0, fmt.Errorf("rotation failed: %w", err)
		}
	}

	n, err = rw.file.Write(p)
	rw.size += int64(n)
	return n, err
}

// Close closes the current log file.
func (rw *RotateWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.file != nil {
		err := rw.file.Close()
		rw.file = nil
		return err
	}
	return nil
}

func (rw *RotateWriter) rotate() error {
	if rw.file != nil {
		rw.file.Close()
		rw.file = nil
	}

	// Rename backup files: stdout.log.4 -> stdout.log.5
	for i := MaxLogFiles - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", rw.filename, i)
		dst := fmt.Sprintf("%s.%d", rw.filename, i+1)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, dst)
		}
	}

	// Rename current log file: stdout.log -> stdout.log.1
	if _, err := os.Stat(rw.filename); err == nil {
		dst := fmt.Sprintf("%s.1", rw.filename)
		if err := os.Rename(rw.filename, dst); err != nil {
			return err
		}
	}

	// Reopen file
	return rw.open()
}

// GetLogPaths returns paths to the stdout and stderr log files for a process.
func GetLogPaths(configDir, name string) (string, string) {
	stdout := filepath.Join(configDir, "logs", name, "stdout.log")
	stderr := filepath.Join(configDir, "logs", name, "stderr.log")
	return stdout, stderr
}

// DeleteLogs deletes all log files for a specific process.
func DeleteLogs(configDir, name string) error {
	logDir := filepath.Join(configDir, "logs", name)
	if err := os.RemoveAll(logDir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// TailFile reads up to the last `numLines` from a file.
// It is implemented efficiently by reading blocks from the end.
func TailFile(filename string, numLines int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	filesize := stat.Size()
	if filesize == 0 {
		return nil, nil
	}

	var (
		lines      []string
		lineBuffer []byte
		readOffset = filesize
		chunkSize  = int64(4096)
	)

	if chunkSize > filesize {
		chunkSize = filesize
	}

	// Read backward in chunks
	for numLines > 0 && readOffset > 0 {
		chunkOffset := readOffset - chunkSize
		if chunkOffset < 0 {
			chunkSize += chunkOffset // decrease size for the very first chunk
			chunkOffset = 0
		}

		buffer := make([]byte, chunkSize)
		_, err := file.ReadAt(buffer, chunkOffset)
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Prepend chunk bytes to our buffer
		lineBuffer = append(buffer, lineBuffer...)
		readOffset = chunkOffset

		// Parse lines from our buffer
		var cursor int
		for i := len(lineBuffer) - 1; i >= 0; i-- {
			if lineBuffer[i] == '\n' {
				// We found a newline
				if i < len(lineBuffer)-1 {
					line := string(lineBuffer[i+1:])
					lines = append([]string{line}, lines...)
					numLines--
					lineBuffer = lineBuffer[:i+1]
					if numLines <= 0 {
						break
					}
				}
				cursor = i
			}
		}

		// If we've reached the beginning of the file and there are remaining bytes
		if readOffset == 0 && len(lineBuffer) > 0 {
			// Trim possible final newline char if it was at index 0 or not found
			line := string(lineBuffer)
			if len(line) > 0 && line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			if len(line) > 0 {
				lines = append([]string{line}, lines...)
			}
			break
		}

		// Retain remainder for next iteration
		if cursor > 0 {
			lineBuffer = lineBuffer[:cursor]
		}
	}

	// Filter empty lines or carriage returns cleanly
	var cleaned []string
	for _, l := range lines {
		// Clean up carriage returns (\r)
		if len(l) > 0 && l[len(l)-1] == '\r' {
			l = l[:len(l)-1]
		}
		cleaned = append(cleaned, l)
	}

	return cleaned, nil
}

// LogFilter reads a log file and returns lines matching a query substring, up to maxLines.
func LogFilter(filename string, query string, maxLines int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var matched []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matchedPattern, _ := filepath.Match(query, line)
		if query == "" || matchedPattern || (len(line) >= len(query) && matchesQuery(line, query)) {
			matched = append(matched, line)
			if len(matched) > maxLines {
				matched = matched[1:] // keep last maxLines
			}
		}
	}

	return matched, scanner.Err()
}

func matchesQuery(line, query string) bool {
	// Simple substring match for simplicity, can also do regex if needed
	// In Go: strings.Contains is fast
	return len(line) >= len(query) && contains(line, query)
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
