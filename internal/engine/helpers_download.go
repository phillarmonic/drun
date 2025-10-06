package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mholt/archives"
	"github.com/phillarmonic/drun/internal/ast"
)

// Domain: Download and Archive Helpers
// This file contains helper methods for file downloads and archive extraction

// downloadFileWithProgress downloads a file using native Go HTTP client with progress tracking
func (e *Engine) downloadFileWithProgress(url, filePath string, headers, auth, options map[string]string) error {
	// Create HTTP client with timeout
	timeout := 30 * time.Second
	if timeoutStr, exists := options["timeout"]; exists {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = duration
		}
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add authentication
	for authType, value := range auth {
		switch authType {
		case "bearer":
			req.Header.Set("Authorization", "Bearer "+value)
		case "basic":
			// Basic auth in format "username:password"
			req.Header.Set("Authorization", "Basic "+value)
		case "token":
			req.Header.Set("Authorization", "Token "+value)
		}
	}

	// Perform request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create parent directories if they don't exist
	if dir := filepath.Dir(filePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	// Create output file
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Get content length for progress tracking
	contentLength := resp.ContentLength

	// Create progress writer
	startTime := time.Now()
	var downloaded int64
	lastUpdate := time.Now()

	// Create a reader that tracks progress
	reader := io.TeeReader(resp.Body, &progressWriter{
		total: contentLength,
		onProgress: func(written int64) {
			downloaded = written
			// Update progress every 100ms to avoid overwhelming output
			if time.Since(lastUpdate) > 100*time.Millisecond || written == contentLength {
				lastUpdate = time.Now()
				e.showDownloadProgress(written, contentLength, time.Since(startTime))
			}
		},
	})

	// Copy data
	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Final progress update
	_, _ = fmt.Fprintf(e.output, "\r\033[K") // Clear line

	// Calculate final stats
	duration := time.Since(startTime)
	speed := float64(downloaded) / duration.Seconds()
	_, _ = fmt.Fprintf(e.output, "   📊 %s in %s (%.2f MB/s)\n",
		formatBytes(downloaded),
		duration.Round(time.Millisecond),
		speed/1024/1024)

	return nil
}

// showDownloadProgress displays download progress with speed and ETA
func (e *Engine) showDownloadProgress(downloaded, total int64, elapsed time.Duration) {
	if total <= 0 {
		// Unknown size, just show downloaded amount
		_, _ = fmt.Fprintf(e.output, "\r   📥 Downloaded: %s", formatBytes(downloaded))
		return
	}

	// Calculate progress percentage
	percent := float64(downloaded) / float64(total) * 100

	// Calculate speed (bytes per second)
	speed := float64(downloaded) / elapsed.Seconds()

	// Calculate ETA
	remaining := total - downloaded
	var eta time.Duration
	if speed > 0 {
		eta = time.Duration(float64(remaining)/speed) * time.Second
	}

	// Create progress bar
	barWidth := 30
	filled := int(float64(barWidth) * percent / 100)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Format output
	_, _ = fmt.Fprintf(e.output, "\r   📥 [%s] %.1f%% | %s/%s | %.2f MB/s | ETA: %s",
		bar,
		percent,
		formatBytes(downloaded),
		formatBytes(total),
		speed/1024/1024,
		formatDuration(eta))
}

// applyFilePermissions applies Unix file permissions based on permission specs
func (e *Engine) applyFilePermissions(path string, permSpecs []ast.PermissionSpec) error {
	// Get current file info
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	currentMode := info.Mode()
	newMode := currentMode

	// Build permission map
	for _, spec := range permSpecs {
		for _, perm := range spec.Permissions {
			for _, target := range spec.Targets {
				// Map permission and target to Unix file mode bits
				var permBits os.FileMode
				switch perm {
				case "read":
					switch target {
					case "user":
						permBits = 0400
					case "group":
						permBits = 0040
					case "others":
						permBits = 0004
					}
				case "write":
					switch target {
					case "user":
						permBits = 0200
					case "group":
						permBits = 0020
					case "others":
						permBits = 0002
					}
				case "execute":
					switch target {
					case "user":
						permBits = 0100
					case "group":
						permBits = 0010
					case "others":
						permBits = 0001
					}
				}

				// Add permission bits
				newMode |= permBits
			}
		}
	}

	// Apply new permissions
	if newMode != currentMode {
		err = os.Chmod(path, newMode)
		if err != nil {
			return fmt.Errorf("failed to chmod: %w", err)
		}
		_, _ = fmt.Fprintf(e.output, "   🔒 Set permissions: %s\n", newMode.String())
	}

	return nil
}

// extractArchive extracts an archive file to the specified directory using the archives library
func (e *Engine) extractArchive(archivePath, extractTo string) error {
	// Create extract directory if it doesn't exist
	err := os.MkdirAll(extractTo, 0755)
	if err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	// Open the archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() { _ = archiveFile.Close() }()

	// Identify the archive format
	format, archiveReader, err := archives.Identify(context.Background(), archivePath, archiveFile)
	if err != nil {
		return fmt.Errorf("failed to identify archive format: %w", err)
	}

	// Check if it's an extractor
	extractor, ok := format.(archives.Extractor)
	if !ok {
		// If it's just compressed (not archived), try to decompress it
		if decompressor, ok := format.(archives.Decompressor); ok {
			return e.decompressFile(decompressor, archiveReader, archivePath, extractTo)
		}
		return fmt.Errorf("format does not support extraction: %s", archivePath)
	}

	// Extract the archive
	handler := func(ctx context.Context, f archives.FileInfo) error {
		// Construct the output path
		outputPath := filepath.Join(extractTo, f.NameInArchive)

		// Handle directories
		if f.IsDir() {
			return os.MkdirAll(outputPath, f.Mode())
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		// Open the file in the archive
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in archive: %w", err)
		}
		defer func() { _ = rc.Close() }()

		// Create the output file
		outFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() { _ = outFile.Close() }()

		// Copy the contents
		if _, err := io.Copy(outFile, rc); err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		return nil
	}

	// Extract all files
	err = extractor.Extract(context.Background(), archiveReader, handler)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	return nil
}

// decompressFile decompresses a single compressed file (not an archive)
func (e *Engine) decompressFile(decompressor archives.Decompressor, reader io.Reader, archivePath, extractTo string) error {
	// Open decompression reader
	rc, err := decompressor.OpenReader(reader)
	if err != nil {
		return fmt.Errorf("failed to open decompressor: %w", err)
	}
	defer func() { _ = rc.Close() }()

	// Determine output filename by removing compression extension
	baseName := filepath.Base(archivePath)
	// Remove common compression extensions
	for _, ext := range []string{".gz", ".bz2", ".xz", ".zst", ".br", ".lz4", ".sz"} {
		if strings.HasSuffix(strings.ToLower(baseName), ext) {
			baseName = strings.TrimSuffix(baseName, ext)
			break
		}
	}

	outputPath := filepath.Join(extractTo, baseName)

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	// Decompress
	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("decompression failed: %w", err)
	}

	return nil
}
