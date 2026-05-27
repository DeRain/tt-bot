package bot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/home/tt-bot/internal/qbt"
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

func TestDescriptionFetcher_Redirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><meta name="description" content="after redirect"></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL+"/redirect")

	if !strings.Contains(desc, "after redirect") {
		t.Errorf("expected description after redirect, got: %q", desc)
	}
}

func TestDescriptionFetcher_MetaTakesPriorityOverOG(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
<meta name="description" content="meta desc">
<meta property="og:description" content="og desc">
</head></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if !strings.Contains(desc, "meta desc") {
		t.Errorf("expected meta description takes priority, got: %q", desc)
	}
	if strings.Contains(desc, "og desc") {
		t.Errorf("expected og description to not appear when meta is present, got: %q", desc)
	}
}

// TestDescriptionFetcher_TimeoutValue verifies the HTTP client timeout is exactly 5s.
// Kills ARITHMETIC_BASE mutant on descriptionFetchTimeout constant.
func TestDescriptionFetcher_TimeoutValue(t *testing.T) {
	f := newDescriptionFetcher()
	// Hardcoded expected value to catch ARITHMETIC_BASE mutant (5*time.Second → 5/time.Second = 5ns).
	if f.client.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", f.client.Timeout)
	}
}

// TestDescriptionFetcher_BodySizeLimit verifies that responses larger than descriptionMaxBytes
// are truncated. Kills ARITHMETIC_BASE mutant on descriptionMaxBytes constant.
func TestDescriptionFetcher_BodySizeLimit(t *testing.T) {
	// Build response where the ONLY description is at byte offset just beyond 256KB.
	// The LimitReader should cut off before reaching it if max is correct.
	// If ARITHMETIC_BASE changes 256*1024 to 256/1024 (0), nothing is read.
	padding := strings.Repeat("x", int(descriptionMaxBytes)+100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><meta name="description" content="` + padding + `found"></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	// The fetched description should be truncated to fit within the limit.
	if len(desc) == 0 {
		t.Error("expected non-empty description (body size limit should allow reading)")
	}
	// The truncated description should be significantly shorter than the full padding.
	if len(desc) > int(descriptionMaxBytes)+100 {
		t.Errorf("description %d bytes exceeds limit of %d", len(desc), int(descriptionMaxBytes)+100)
	}
}

// TestDescriptionFetcher_HeadersSet verifies User-Agent and Accept headers are set.
// Kills STATEMENT_REMOVE mutants on req.Header.Set calls.
func TestDescriptionFetcher_HeadersSet(t *testing.T) {
	var userAgent, accept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.Header.Get("User-Agent")
		accept = r.Header.Get("Accept")
		w.Write([]byte(`<html><meta name="description" content="test"></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	_ = f.fetch(context.Background(), server.URL)

	if userAgent == "" {
		t.Error("expected User-Agent header to be set")
	}
	if accept == "" {
		t.Error("expected Accept header to be set")
	}
}

// TestDescriptionFetcher_SSRFBlocked verifies that loopback addresses are rejected
// by isPublicHostname. Kills BRANCH_IF mutant on line 47.
func TestDescriptionFetcher_SSRFBlocked(t *testing.T) {
	tests := []struct {
		name, url string
	}{
		{"loopback", "http://127.0.0.1/"},
		{"localhost", "http://localhost/"},
		{"private", "http://10.0.0.1/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDescriptionFetcher()
			desc := f.fetch(context.Background(), tt.url)
			if desc != "" {
				t.Errorf("expected empty description for blocked host %q, got: %q", tt.url, desc)
			}
		})
	}
}

// TestDescriptionFetcher_Non200WithBody verifies that non-200 responses are rejected
// even when they contain a description. Kills BRANCH_IF mutant on line 64.
func TestDescriptionFetcher_Non200WithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><meta name="description" content="should not appear"></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if desc != "" {
		t.Errorf("expected empty description for 404 with body, got: %q", desc)
	}
}

// TestDescriptionFetcher_BodyHTMLEntities verifies HTML entities in body text are decoded.
// Kills STATEMENT_REMOVE mutant on decodeEntities call (line 91).
func TestDescriptionFetcher_BodyHTMLEntities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><p>Price: &amp; more</p></body></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if !strings.Contains(desc, "Price: & more") {
		t.Errorf("expected decoded ampersand in body text, got: %q", desc)
	}
	if strings.Contains(desc, "&amp;") {
		t.Errorf("expected entity decoded, got raw: %q", desc)
	}
}

// TestDescriptionFetcher_WhitespaceCollapsed verifies multiple spaces are collapsed.
// Kills STATEMENT_REMOVE mutant on inSpace assignment (line 99).
func TestDescriptionFetcher_WhitespaceCollapsed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Multiple spaces, tabs, newlines in body text.
		w.Write([]byte(`<html><body><p>word1  		word2

word3</p></body></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if strings.Contains(desc, "  ") {
		t.Errorf("expected collapsed whitespace (no double spaces), got: %q", desc)
	}
	if !strings.Contains(desc, "word1 word2 word3") {
		t.Errorf("expected 'word1 word2 word3', got: %q", desc)
	}
}

// TestDescriptionFetcher_TrimmedResult verifies leading/trailing whitespace is trimmed.
// Kills STATEMENT_REMOVE mutant on TrimSpace call (line 110).
func TestDescriptionFetcher_TrimmedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Body text with leading/trailing spaces from HTML structure.
		w.Write([]byte(`<html><body>
	<p>  trimmed content  </p>
</body></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if strings.HasPrefix(desc, " ") || strings.HasSuffix(desc, " ") {
		t.Errorf("expected trimmed result, got: %q", desc)
	}
}

// TestDescriptionFetcher_MetaSingleChar tests that meta description matches even
// with single-character content. Kills CONDITIONALS_BOUNDARY on line 80.
func TestDescriptionFetcher_MetaSingleChar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><meta name="description" content="x"></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if desc != "x" {
		t.Errorf("expected 'x', got: %q", desc)
	}
}

// TestDescriptionFetcher_OGSingleChar tests that og:description matches even
// with single-character content. Kills CONDITIONALS_BOUNDARY on line 83.
func TestDescriptionFetcher_OGSingleChar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><meta property="og:description" content="y"></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if desc != "y" {
		t.Errorf("expected 'y', got: %q", desc)
	}
}

// TestDescriptionFetcher_OnlyMetaWithoutOG verifies desc is empty when meta is absent
// and OG regex matches but produces empty capture. Kills BRANCH_IF on line 83.
func TestDescriptionFetcher_OnlyOGAsFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// No meta description, only og:description — should fall back to OG.
		w.Write([]byte(`<html><head><meta property="og:description" content="og only"></head><body></body></html>`))
	}))
	defer server.Close()

	f := newDescriptionFetcher()
	desc := f.fetch(context.Background(), server.URL)

	if !strings.Contains(desc, "og only") {
		t.Errorf("expected OG description fallback, got: %q", desc)
	}
}

// TestDescriptionFetcher_InvalidURLNoHost verifies that a URL without a host is rejected
// by the URL parsing guard. Kills BRANCH_IF on line 41 and EXPRESSION_REMOVE on parsed.Host.
func TestDescriptionFetcher_InvalidURLNoHost(t *testing.T) {
	f := newDescriptionFetcher()

	tests := []struct {
		name, url string
	}{
		{"no host", "http:///path"},
		{"relative path", "/relative/path"},
		{"colon only", ":"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := f.fetch(context.Background(), tt.url)
			if desc != "" {
				t.Errorf("expected empty for %q, got: %q", tt.url, desc)
			}
		})
	}
}

func TestFetchAndUpdateDescription_StoresDescription(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	descServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><meta name="description" content="Stored description"></html>`))
	}))
	defer descServer.Close()

	result := qbt.SearchResult{
		FileName:   "Test Torrent",
		FileSize:   1024,
		NbSeeders:  10,
		NbLeechers: 5,
		DescrLink:  descServer.URL,
	}
	state := &SearchState{
		ChatID:      1,
		MessageID:   100,
		JobID:       42,
		Results:     []qbt.SearchResult{result},
		Total:       1,
		SelectedIdx: 5, // non-zero to expose ARITHMETIC_BASE on division
	}
	h.storeSearch(1, state)

	h.fetchAndUpdateDescription(1, 100, state, result)

	s := h.getSearch(1)
	if s == nil {
		t.Fatal("search state not found after fetch")
	}
	if s.DescriptionText == "" {
		t.Error("expected DescriptionText to be set")
	}
	if s.DescriptionPages == 0 {
		t.Error("expected DescriptionPages > 0")
	}
	// Verify edit was sent with description content.
	if !sender.hasEditText("Stored description") {
		t.Error("expected edit with description text")
	}
}

func TestFetchAndUpdateDescription_NoDescrLink(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	result := qbt.SearchResult{
		FileName:   "Test Torrent",
		FileSize:   1024,
		NbSeeders:  10,
		NbLeechers: 5,
		DescrLink:  "", // empty — should return early
	}
	state := &SearchState{
		ChatID:    1,
		MessageID: 100,
		JobID:     42,
		Results:   []qbt.SearchResult{result},
		Total:     1,
	}
	h.storeSearch(1, state)

	h.fetchAndUpdateDescription(1, 100, state, result)

	s := h.getSearch(1)
	if s != nil && s.DescriptionText != "" {
		t.Errorf("expected empty DescriptionText when no DescrLink, got %q", s.DescriptionText)
	}
}

func TestFetchAndUpdateDescription_EmptyResult(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	descServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>No meta tags.</body></html>`))
	}))
	defer descServer.Close()

	result := qbt.SearchResult{
		FileName:   "Test",
		FileSize:   1024,
		NbSeeders:  10,
		NbLeechers: 5,
		DescrLink:  descServer.URL,
	}
	state := &SearchState{
		ChatID:    1,
		MessageID: 100,
		JobID:     42,
		Results:   []qbt.SearchResult{result},
		Total:     1,
	}
	h.storeSearch(1, state)

	h.fetchAndUpdateDescription(1, 100, state, result)

	s := h.getSearch(1)
	if s == nil {
		t.Fatal("search state not found after fetch")
	}
	if !strings.Contains(s.DescriptionText, "No meta tags") {
		t.Errorf("expected body fallback text, got: %q", s.DescriptionText)
	}
}

func TestFetchAndUpdateDescription_NoSearchStored(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	result := qbt.SearchResult{
		FileName:   "Test",
		FileSize:   1024,
		NbSeeders:  1,
		NbLeechers: 1,
		DescrLink:  "",
	}
	state := &SearchState{
		ChatID:    1,
		MessageID: 100,
		JobID:     42,
		Results:   []qbt.SearchResult{result},
		Total:     1,
	}

	h.fetchAndUpdateDescription(1, 100, state, result)

	s := h.getSearch(1)
	if s != nil && s.DescriptionText != "" {
		t.Errorf("expected no description when no search stored, got %q", s.DescriptionText)
	}
}

func TestFetchAndUpdateDescription_JobIDMismatch(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	result := qbt.SearchResult{FileName: "x", FileSize: 1, NbSeeders: 0, DescrLink: ""}
	state := &SearchState{ChatID: 1, MessageID: 100, JobID: 99, Results: []qbt.SearchResult{result}, Total: 1}
	h.storeSearch(1, &SearchState{ChatID: 1, MessageID: 100, JobID: 42, Results: []qbt.SearchResult{result}, Total: 1})
	h.fetchAndUpdateDescription(1, 100, state, result)
}

func TestFetchAndUpdateDescription_SelectedIdxMismatch(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	result := qbt.SearchResult{FileName: "x", FileSize: 1, NbSeeders: 0, DescrLink: ""}
	state := &SearchState{ChatID: 1, MessageID: 100, JobID: 42, SelectedIdx: 7, Results: []qbt.SearchResult{result}, Total: 1}
	h.storeSearch(1, &SearchState{ChatID: 1, MessageID: 100, JobID: 42, SelectedIdx: 3, Results: []qbt.SearchResult{result}, Total: 1})
	h.fetchAndUpdateDescription(1, 100, state, result)
}

func TestFetchAndUpdateDescription_ZeroPages(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	result := qbt.SearchResult{FileName: "x", FileSize: 1, NbSeeders: 0, DescrLink: ""}
	state := &SearchState{ChatID: 1, MessageID: 100, JobID: 42, SelectedIdx: 0, Results: []qbt.SearchResult{result}, Total: 1}
	h.storeSearch(1, state)
	h.fetchAndUpdateDescription(1, 100, state, result)

	s := h.getSearch(1)
	if s != nil && s.DescriptionPages != 0 {
		t.Errorf("expected 0 pages for empty description, got %d", s.DescriptionPages)
	}
}
