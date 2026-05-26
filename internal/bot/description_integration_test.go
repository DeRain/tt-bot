//go:build integration

package bot

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// startTestServer starts a temporary HTTP server on a random port.
func startTestServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return "http://" + listener.Addr().String()
}

func TestIntegration_DescriptionFetcher_MetaDescription(t *testing.T) {
	url := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <meta name="description" content="Ubuntu 24.04 LTS (Noble Numbat) Daily Build">
  <title>Ubuntu 24.04</title>
</head>
<body>
  <h1>Ubuntu 24.04 LTS</h1>
  <p>Download the latest daily build of Ubuntu 24.04.</p>
</body>
</html>`))
	})

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), url)

	if !strings.Contains(desc, "Ubuntu 24.04 LTS") {
		t.Errorf("expected meta description, got: %q", desc)
	}
	if len(desc) > maxDescriptionChars {
		t.Errorf("description too long: %d chars", len(desc))
	}
}

func TestIntegration_DescriptionFetcher_OGFallback(t *testing.T) {
	url := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
<head>
  <meta property="og:description" content="OpenGraph fallback text.">
</head>
<body><p>Body text that should not appear when OG is present.</p></body>
</html>`))
	})

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), url)

	if !strings.Contains(desc, "OpenGraph fallback text") {
		t.Errorf("expected OG description, got: %q", desc)
	}
}

func TestIntegration_DescriptionFetcher_BodyExtraction(t *testing.T) {
	url := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
<head><title>No Meta</title></head>
<body>
  <p>First paragraph of body text.</p>
  <p>Second paragraph with <b>bold</b> and <i>italic</i>.</p>
</body>
</html>`))
	})

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), url)

	if !strings.Contains(desc, "First paragraph") {
		t.Errorf("expected body text, got: %q", desc)
	}
	if !strings.Contains(desc, "Second paragraph") {
		t.Errorf("expected second paragraph, got: %q", desc)
	}
	if strings.Contains(desc, "<b>") || strings.Contains(desc, "<i>") {
		t.Errorf("expected HTML tags stripped, got: %q", desc)
	}
}

func TestIntegration_DescriptionFetcher_Timeout(t *testing.T) {
	url := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
	})

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), url)

	if desc != "" {
		t.Errorf("expected empty description on timeout, got: %q", desc)
	}
}

func TestIntegration_DescriptionFetcher_LargeResponse(t *testing.T) {
	largeBody := strings.Repeat("x", 300*1024) // 300KB, over 256KB limit
	url := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><meta name=\"description\" content=\"short\"><body>" + largeBody + "</body></html>"))
	})

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), url)

	if !strings.Contains(desc, "short") {
		t.Errorf("expected meta description from large response, got: %q", desc)
	}
	// Description should NOT be truncated — pagination handles long text.
	if strings.HasSuffix(desc, "...") {
		t.Errorf("description should not be truncated; pagination handles long text. got: %q", desc)
	}
}
