package http_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	httpinfra "github.com/dydanz/codeburn-watcher/infrastructure/http"
)

func TestHttpExportTransport_Push(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(1 << 20)
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("form file: %v", err)
			w.WriteHeader(400)
			return
		}
		received, _ = io.ReadAll(f)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	transport := httpinfra.NewHttpTransport()
	body := []byte(`{"hello":"world"}`)
	if err := transport.Push(context.Background(), "test.json", body, srv.URL); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if string(received) != string(body) {
		t.Errorf("received %q, want %q", received, body)
	}
}

func TestFilesystemExportTransport_Push(t *testing.T) {
	dir := t.TempDir()
	transport := httpinfra.FilesystemExportTransport{}
	body := []byte(`{"test":true}`)

	if err := transport.Push(context.Background(), "export.json", body, dir); err != nil {
		t.Fatalf("Push: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "export.json"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != string(body) {
		t.Errorf("file content = %q, want %q", data, body)
	}
}

func TestCompositeExportTransport_Dispatch(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	defer srv.Close()

	composite := httpinfra.CompositeExportTransport{
		HTTP: httpinfra.NewHttpTransport(),
		FS:   httpinfra.FilesystemExportTransport{},
	}
	// http URL → HTTP transport
	_ = composite.Push(context.Background(), "f.json", []byte("{}"), srv.URL)
	if !called {
		t.Error("expected HTTP transport to be called for http:// URL")
	}
}
