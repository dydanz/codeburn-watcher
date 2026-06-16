package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HttpExportTransport sends exports via HTTP POST.
// Uses multipart/form-data when Content-Type negotiation is needed.
type HttpExportTransport struct {
	Client *http.Client
}

// NewHttpTransport returns an HttpExportTransport with a 30-second timeout.
func NewHttpTransport() HttpExportTransport {
	return HttpExportTransport{Client: &http.Client{Timeout: 30 * time.Second}}
}

// Push POSTs body as multipart/form-data with the filename as the "file" field.
func (t HttpExportTransport) Push(ctx context.Context, filename string, body []byte, url string) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(fw, bytes.NewReader(body)); err != nil {
		return fmt.Errorf("write form file: %w", err)
	}
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := t.Client.Do(req)
	if err != nil {
		return fmt.Errorf("http push: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http push: server returned %d", resp.StatusCode)
	}
	return nil
}

// FetchConfig retrieves a raw team config string from a URL.
func (t HttpExportTransport) FetchConfig(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := t.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch config: server returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read config body: %w", err)
	}
	return string(data), nil
}

// FilesystemExportTransport copies exports to a local path destination.
type FilesystemExportTransport struct{}

// Push writes body to filepath.Join(destination, filename).
func (FilesystemExportTransport) Push(_ context.Context, filename string, body []byte, destination string) error {
	if err := os.MkdirAll(destination, 0750); err != nil {
		return fmt.Errorf("mkdir %s: %w", destination, err)
	}
	return os.WriteFile(filepath.Join(destination, filename), body, 0644)
}

// FetchConfig reads a local file at url (treated as a path).
func (FilesystemExportTransport) FetchConfig(_ context.Context, url string) (string, error) {
	data, err := os.ReadFile(url)
	if err != nil {
		return "", fmt.Errorf("read config file: %w", err)
	}
	return string(data), nil
}

// CompositeExportTransport dispatches to HTTP or FS based on the destination scheme.
type CompositeExportTransport struct {
	HTTP HttpExportTransport
	FS   FilesystemExportTransport
}

func (c CompositeExportTransport) Push(ctx context.Context, filename string, body []byte, destination string) error {
	if strings.HasPrefix(destination, "http") {
		return c.HTTP.Push(ctx, filename, body, destination)
	}
	return c.FS.Push(ctx, filename, body, destination)
}

func (c CompositeExportTransport) FetchConfig(ctx context.Context, url string) (string, error) {
	if strings.HasPrefix(url, "http") {
		return c.HTTP.FetchConfig(ctx, url)
	}
	return c.FS.FetchConfig(ctx, url)
}
