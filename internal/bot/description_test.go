package bot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDescriptionFetcher_MetaDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><meta name="description" content="Ubuntu 24.04 LTS ISO download"></head><body></body></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if !strings.Contains(desc, "Ubuntu 24.04 LTS ISO download") {
		t.Errorf("expected meta description, got: %q", desc)
	}
}

func TestDescriptionFetcher_OGDescription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><meta property="og:description" content="OpenGraph desc"></head><body></body></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if !strings.Contains(desc, "OpenGraph desc") {
		t.Errorf("expected og:description, got: %q", desc)
	}
}

func TestDescriptionFetcher_BodyFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><p>No meta tags here.</p><p>Just body text.</p></body></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if !strings.Contains(desc, "No meta tags here") {
		t.Errorf("expected body text, got: %q", desc)
	}
	if !strings.Contains(desc, "Just body text") {
		t.Errorf("expected second paragraph, got: %q", desc)
	}
}

func TestDescriptionFetcher_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(``))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if desc != "" {
		t.Errorf("expected empty description, got: %q", desc)
	}
}

func TestDescriptionFetcher_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if desc != "" {
		t.Errorf("expected empty description on 403, got: %q", desc)
	}
}

func TestDescriptionFetcher_InvalidURL(t *testing.T) {
	f := newDescriptionFetcher()

	tests := []struct {
		name, url string
	}{
		{"empty", ""},
		{"no scheme", "example.com/torrent"},
		{"ftp", "ftp://example.com/file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := f.fetch(context.Background(), tt.url)
			if desc != "" {
				t.Errorf("expected empty description for %q, got: %q", tt.url, desc)
			}
		})
	}
}

func TestDescriptionFetcher_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><meta name="description" content="test"></html>`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := newDescriptionFetcher()
	desc := f.fetch(ctx, server.URL)

	if desc != "" {
		t.Errorf("expected empty description for canceled context, got: %q", desc)
	}
}

func TestDescriptionFetcher_HTMLTagsStripped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><p>Text <b>with</b> <i>tags</i></p></body></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if strings.Contains(desc, "<b>") || strings.Contains(desc, "<i>") {
		t.Errorf("expected HTML tags stripped, got: %q", desc)
	}
}

func TestDescriptionFetcher_LongDescriptionPreserved(t *testing.T) {
	longText := strings.Repeat("x", 5000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><meta name="description" content="` + longText + `"></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if !strings.Contains(desc, longText) {
		t.Errorf("expected full description preserved (pagination handles long text), got %d chars", len(desc))
	}
	if strings.HasSuffix(desc, "...") {
		t.Errorf("expected no truncation suffix, got: %q", desc)
	}
}

func TestDescriptionFetcher_HTMLEntitiesDecoded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><meta name="description" content="A &amp; B &lt; C &gt; D"></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if strings.Contains(desc, "&amp;") {
		t.Errorf("expected HTML entities decoded, got: %q", desc)
	}
	if !strings.Contains(desc, "A & B") {
		t.Errorf("expected decoded ampersand, got: %q", desc)
	}
}
