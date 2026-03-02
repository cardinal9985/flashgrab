package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ProgressFunc is called periodically during a download with the number of
// bytes transferred so far and the total size (0 if unknown).
type ProgressFunc func(downloaded, total int64)

// Manager handles downloading files to disk with progress reporting.
type Manager struct {
	client *http.Client
	dir    string
}

// New creates a download manager that saves files to dir.
func New(dir string) *Manager {
	return &Manager{
		client: &http.Client{
			Timeout: 0, // no overall timeout—we use idle detection instead
			Transport: &http.Transport{
				ResponseHeaderTimeout: 30 * time.Second,
				IdleConnTimeout:       60 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				switch req.URL.Scheme {
				case "http", "https":
					return nil
				default:
					return fmt.Errorf("unsafe redirect to %s", req.URL.Scheme)
				}
			},
		},
		dir: dir,
	}
}

// Result holds information about a completed download.
type Result struct {
	Path     string // full path to the saved file
	Filename string // just the filename
	Size     int64  // bytes written
	Existed  bool   // true if the file already existed and was skipped
}

// Fetch downloads the file at fileURL and saves it to the configured directory
// using the given filename. The onProgress callback fires roughly every 100ms
// during the transfer. If a file with the same name already exists, the
// download is skipped.
func (m *Manager) Fetch(fileURL, filename string, onProgress ProgressFunc) (*Result, error) {
	dest := filepath.Join(m.dir, filename)

	// Don't re-download files that already exist.
	if info, err := os.Stat(dest); err == nil {
		return &Result{
			Path:     dest,
			Filename: filename,
			Size:     info.Size(),
			Existed:  true,
		}, nil
	}

	// Make sure the target directory exists.
	if err := os.MkdirAll(m.dir, 0755); err != nil {
		return nil, fmt.Errorf("creating download dir: %w", err)
	}

	resp, err := m.client.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("requesting file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d %s", resp.StatusCode, resp.Status)
	}

	total := resp.ContentLength

	// Write to a temp file first, then rename. This prevents partial files
	// from cluttering the download directory if something goes wrong.
	tmp, err := os.CreateTemp(m.dir, ".flashgrab-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpPath) // no-op if rename succeeded
	}()

	// Wrap the response body in a progress reporter.
	reader := &progressReader{
		reader:     resp.Body,
		total:      total,
		onProgress: onProgress,
		interval:   100 * time.Millisecond,
	}

	written, err := io.Copy(tmp, reader)
	if err != nil {
		return nil, fmt.Errorf("downloading: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return nil, fmt.Errorf("saving file: %w", err)
	}

	return &Result{
		Path:     dest,
		Filename: filename,
		Size:     written,
	}, nil
}

// progressReader wraps an io.Reader and calls a callback at regular intervals
// with download progress.
type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	onProgress ProgressFunc
	interval   time.Duration
	lastReport time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.downloaded += int64(n)

	if pr.onProgress != nil {
		now := time.Now()
		if now.Sub(pr.lastReport) >= pr.interval || err == io.EOF {
			pr.onProgress(pr.downloaded, pr.total)
			pr.lastReport = now
		}
	}

	return n, err
}
