// Package bot contains TDD-red tests for download safety functions.
//
// These tests reference functions (isPublicHostname, downloadSearchTorrent,
// newDownloadClient) that DO NOT YET EXIST. This file will fail to compile
// until they are implemented in a corresponding download.go — this is the
// expected TDD red phase outcome.
package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func init() {
	// Allow loopback in tests — httptest servers bind to 127.0.0.1.
	ssrfCheck = func(host string) error { return nil }
}

// =========================================================================
// isPublicHostname tests
// =========================================================================

// Design note for isPublicHostname:
//
// To enable testing without a DNS dependency, the function should first try
// to parse the input as an IP address. If it is a valid IP literal (v4 or
// v6), it should check the address directly against private/reserved/link-
// local range tables without making any network calls.
//
// For non-IP hostnames (e.g. "example.com") the function may need DNS
// resolution. To keep those paths testable, consider accepting an optional
// resolver interface (such as a *net.Resolver parameter or a functional
// option) that tests can replace with a stub.
//
// The tests below exercise only the IP-literal path since it requires no
// external dependency.

func TestIsPublicHostname_RejectsLoopback(t *testing.T) {
	hosts := []string{"127.0.0.1", "127.0.0.2", "::1"}
	for _, h := range hosts {
		t.Run(h, func(t *testing.T) {
			if err := isPublicHostname(h); err == nil {
				t.Errorf("isPublicHostname(%q) = nil, want error", h)
			}
		})
	}
}

func TestIsPublicHostname_RejectsPrivate(t *testing.T) {
	hosts := []string{
		"10.0.0.1", "10.255.255.255",
		"172.16.0.1", "172.31.255.255",
		"192.168.0.1", "192.168.255.255",
	}
	for _, h := range hosts {
		t.Run(h, func(t *testing.T) {
			if err := isPublicHostname(h); err == nil {
				t.Errorf("isPublicHostname(%q) = nil, want error", h)
			}
		})
	}
}

func TestIsPublicHostname_RejectsLinkLocal(t *testing.T) {
	hosts := []string{"169.254.0.1", "169.254.255.254"}
	for _, h := range hosts {
		t.Run(h, func(t *testing.T) {
			if err := isPublicHostname(h); err == nil {
				t.Errorf("isPublicHostname(%q) = nil, want error", h)
			}
		})
	}
}

func TestIsPublicHostname_AcceptsPublic(t *testing.T) {
	hosts := []string{"8.8.8.8", "1.1.1.1"}
	for _, h := range hosts {
		t.Run(h, func(t *testing.T) {
			if err := isPublicHostname(h); err != nil {
				t.Errorf("isPublicHostname(%q) = %v, want nil", h, err)
			}
		})
	}
}

func TestIsPublicHostname_InvalidHostname(t *testing.T) {
	hosts := []string{"", "not a valid hostname with spaces!"}
	for _, h := range hosts {
		t.Run(h, func(t *testing.T) {
			if err := isPublicHostname(h); err == nil {
				t.Errorf("isPublicHostname(%q) = nil, want error", h)
			}
		})
	}
}

func TestIsPublicHostname_RejectsUnspecified(t *testing.T) {
	hosts := []string{"0.0.0.0", "::"}
	for _, h := range hosts {
		t.Run(h, func(t *testing.T) {
			if err := isPublicHostname(h); err == nil {
				t.Errorf("isPublicHostname(%q) = nil, want error", h)
			}
		})
	}
}

func TestIsPublicHostname_AcceptsHostname(t *testing.T) {
	// Valid hostnames (non-IP) should be accepted — no DNS lookup needed yet.
	hosts := []string{"example.com", "tracker.example.org"}
	for _, h := range hosts {
		t.Run(h, func(t *testing.T) {
			if err := isPublicHostname(h); err != nil {
				t.Errorf("isPublicHostname(%q) = %v, want nil", h, err)
			}
		})
	}
}

// =========================================================================
// downloadSearchTorrent tests
// =========================================================================
//
// downloadSearchTorrent(ctx, client, url) wraps a raw HTTP download with
// safety checks:
//   - Content-Type must be application/x-bittorrent (reject text/html etc.)
//   - Response must be ≤10 MB
//   - HTTP status must be 200 OK
//   - Timeout is enforced via client.Timeout
//   - Redirects are limited (client.CheckRedirect)
//   - HTTPS→HTTP scheme downgrade is rejected

func TestDownloadSearchTorrent_Success(t *testing.T) {
	torrentData := []byte("d8:announce...")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write(torrentData)
	}))
	defer srv.Close()

	client := srv.Client()
	data, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(data, torrentData) {
		t.Errorf("got %q, want %q", data, torrentData)
	}
}

func TestDownloadSearchTorrent_HTMLRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>not a torrent</body></html>"))
	}))
	defer srv.Close()

	client := srv.Client()
	_, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err == nil {
		t.Error("expected error for text/html Content-Type, got nil")
	}
}

func TestDownloadSearchTorrent_TooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		// Write 11 MB (exceeds a 10 MB safety limit).
		chunk := make([]byte, 1024*1024) // 1 MB
		for i := 0; i < 11; i++ {
			_, _ = w.Write(chunk)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	_, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err == nil {
		t.Error("expected error for >10 MB response, got nil")
	}
}

func TestDownloadSearchTorrent_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer srv.Close()

	client := srv.Client()
	_, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err == nil {
		t.Error("expected error for HTTP 404, got nil")
	}
}

func TestDownloadSearchTorrent_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("too late"))
	}))
	defer srv.Close()

	// Client with aggressive timeout that should fire before the handler.
	srvClient := srv.Client()
	client := &http.Client{
		Timeout:   50 * time.Millisecond,
		Transport: srvClient.Transport,
	}
	_, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err == nil {
		t.Error("expected error for request timeout, got nil")
	}
}

func TestDownloadSearchTorrent_TooManyRedirects(t *testing.T) {
	var redirectCount int
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount <= 6 {
			http.Redirect(w, r, srv.URL, http.StatusFound)
		} else {
			w.Header().Set("Content-Type", "application/x-bittorrent")
			_, _ = w.Write([]byte("d8:announce..."))
		}
	}))
	defer srv.Close()

	client := newDownloadClient()
	_, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err == nil {
		t.Error("expected error for redirect chain of 6, got nil")
	} else if !strings.Contains(err.Error(), "redirect") && !strings.Contains(err.Error(), "stopped after") {
		t.Errorf("expected redirect limit error, got: %v", err)
	}
}

func TestDownloadSearchTorrent_HTTPSDowngrade(t *testing.T) {
	// Plain HTTP server (the eventual redirect target).
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write([]byte("d8:announce..."))
	}))
	defer httpSrv.Close()

	// TLS server that redirects to the plain HTTP URL (scheme downgrade).
	httpsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, httpSrv.URL, http.StatusFound)
	}))
	defer httpsSrv.Close()

	client := httpsSrv.Client()
	_, err := downloadSearchTorrent(context.Background(), client, httpsSrv.URL)
	if err == nil {
		t.Error("expected error for HTTPS→HTTP scheme downgrade, got nil")
	}
}

func TestDownloadSearchTorrent_BadURL(t *testing.T) {
	client := &http.Client{}
	_, err := downloadSearchTorrent(context.Background(), client, "://invalid")
	if err == nil {
		t.Error("expected parse error for malformed URL, got nil")
	}
}

func TestDownloadSearchTorrent_UnsupportedScheme(t *testing.T) {
	client := &http.Client{}
	_, err := downloadSearchTorrent(context.Background(), client, "ftp://example.com/torrent")
	if err == nil {
		t.Error("expected error for ftp:// scheme, got nil")
	} else if !strings.Contains(err.Error(), "unsupported scheme") {
		t.Errorf("expected 'unsupported scheme' error, got: %v", err)
	}
}

func TestDownloadSearchTorrent_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{}
	_, err := downloadSearchTorrent(ctx, client, "http://example.com/torrent")
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestDownloadSearchTorrent_SSRFRejected(t *testing.T) {
	orig := ssrfCheck
	ssrfCheck = isPublicHostname
	defer func() { ssrfCheck = orig }()

	client := &http.Client{}
	_, err := downloadSearchTorrent(context.Background(), client, "http://10.0.0.1/torrent")
	if err == nil {
		t.Error("expected SSRF rejection for private IP, got nil")
	} else if !strings.Contains(err.Error(), "unsafe hostname") {
		t.Errorf("expected 'unsafe hostname' error, got: %v", err)
	}
}

// =========================================================================
// newDownloadClient tests
// =========================================================================

func TestNewDownloadClient_Timeout(t *testing.T) {
	client := newDownloadClient()
	if client.Timeout != 15*time.Second {
		t.Errorf("newDownloadClient().Timeout = %v, want 15s", client.Timeout)
	}
}

func TestNewDownloadClient_RedirectPolicy(t *testing.T) {
	client := newDownloadClient()
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}

	// Build a via slice where every element is the same request — only
	// the slice length matters for the redirect-count check.
	makeVia := func(n int) []*http.Request {
		via := make([]*http.Request, n)
		for i := range via {
			via[i] = req
		}
		return via
	}

	// 4 prior redirects → should be allowed (this is the 5th redirect attempt).
	if err := client.CheckRedirect(req, makeVia(4)); err != nil {
		t.Errorf("expected nil for 4 prior redirects, got: %v", err)
	}

	// 5 prior redirects → must be rejected (this is the 6th redirect attempt).
	if err := client.CheckRedirect(req, makeVia(5)); err == nil {
		t.Error("expected error for 5 prior redirects (6th redirect), got nil")
	}
}

// =========================================================================
// HTTPS success path for downloadSearchTorrent
// =========================================================================

func TestDownloadSearchTorrent_HttpsSuccess(t *testing.T) {
	torrentData := []byte("d8:announce...")

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write(torrentData)
	}))
	defer srv.Close()

	client := srv.Client()
	data, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(data, torrentData) {
		t.Errorf("got %q, want %q", data, torrentData)
	}
}

// =========================================================================
// newDownloadClient CheckRedirect — scheme downgrade and boundary
// =========================================================================

func TestNewDownloadClient_DowngradeRejected(t *testing.T) {
	client := newDownloadClient()

	httpsReq, err := http.NewRequest(http.MethodGet, "https://example.com/initial", nil)
	if err != nil {
		t.Fatalf("building https request: %v", err)
	}
	httpReq, err := http.NewRequest(http.MethodGet, "http://example.com/redirect", nil)
	if err != nil {
		t.Fatalf("building http request: %v", err)
	}

	// HTTPS→HTTP downgrade → must be rejected.
	via := []*http.Request{httpsReq}
	err = client.CheckRedirect(httpReq, via)
	if err == nil {
		t.Error("expected error for HTTPS→HTTP downgrade, got nil")
	}
}

func TestNewDownloadClient_DowngradeApproved(t *testing.T) {
	client := newDownloadClient()

	httpsReq1, err := http.NewRequest(http.MethodGet, "https://example.com/initial", nil)
	if err != nil {
		t.Fatalf("building first https request: %v", err)
	}
	httpsReq2, err := http.NewRequest(http.MethodGet, "https://example.com/redirect", nil)
	if err != nil {
		t.Fatalf("building second https request: %v", err)
	}

	// HTTPS→HTTPS same-scheme → must be allowed.
	via := []*http.Request{httpsReq1}
	err = client.CheckRedirect(httpsReq2, via)
	if err != nil {
		t.Errorf("expected nil for HTTPS→HTTPS redirect, got: %v", err)
	}
}

func TestNewDownloadClient_HttpToHttpOk(t *testing.T) {
	client := newDownloadClient()

	httpReq1, err := http.NewRequest(http.MethodGet, "http://example.com/initial", nil)
	if err != nil {
		t.Fatalf("building first http request: %v", err)
	}
	httpReq2, err := http.NewRequest(http.MethodGet, "http://example.com/redirect", nil)
	if err != nil {
		t.Fatalf("building second http request: %v", err)
	}

	// HTTP→HTTP same-scheme → must be allowed.
	via := []*http.Request{httpReq1}
	err = client.CheckRedirect(httpReq2, via)
	if err != nil {
		t.Errorf("expected nil for HTTP→HTTP redirect, got: %v", err)
	}
}

// =========================================================================
// downloadSearchTorrent — body read error via custom transport
// =========================================================================

// errorBodyReader returns a constant error on every Read call.
type errorBodyReader struct {
	read bool
}

func (r *errorBodyReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, io.EOF
	}
	r.read = true
	return 0, fmt.Errorf("simulated body read error")
}

func (r *errorBodyReader) Close() error { return nil }

// errorBodyTransport returns a valid HTTP response whose body fails on read.
type errorBodyTransport struct{}

func (t *errorBodyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/x-bittorrent"},
		},
		Body: &errorBodyReader{},
	}, nil
}

func TestDownloadSearchTorrent_BodyReadError(t *testing.T) {
	client := &http.Client{
		Transport: &errorBodyTransport{},
	}
	_, err := downloadSearchTorrent(context.Background(), client, "http://example.com/t")
	if err == nil {
		t.Error("expected body read error, got nil")
	}
}

// =========================================================================
// downloadSearchTorrent — maxSize boundary tests
// =========================================================================

func TestDownloadSearchTorrent_ExactMaxSize(t *testing.T) {
	// maxSize = 10 * 1024 * 1024 = 10485760
	const maxSize = 10 * 1024 * 1024
	data := make([]byte, maxSize)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	client := srv.Client()
	got, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err != nil {
		t.Fatalf("expected success for %d bytes, got: %v", maxSize, err)
	}
	if len(got) != maxSize {
		t.Errorf("got %d bytes, want %d", len(got), maxSize)
	}
}

func TestDownloadSearchTorrent_JustOverMaxSize(t *testing.T) {
	// maxSize = 10 * 1024 * 1024 = 10485760
	const maxSize = 10 * 1024 * 1024
	data := make([]byte, maxSize+1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	client := srv.Client()
	_, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err == nil {
		t.Error("expected error for response exceeding maxSize, got nil")
	}
}

// =========================================================================
// downloadSearchTorrent — HTTPS→HTTPS redirect (no downgrade)
// =========================================================================

func TestDownloadSearchTorrent_HttpsToHttpsRedirect(t *testing.T) {
	// TLS server that redirects to a final TLS endpoint.
	var finalURL string
	redirectSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalURL, http.StatusFound)
	}))
	defer redirectSrv.Close()

	finalSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write([]byte("d8:announce..."))
	}))
	defer finalSrv.Close()

	finalURL = finalSrv.URL

	client := redirectSrv.Client()
	// Make the client trust both servers by using redirectSrv.Client() which
	// trusts all TLS certs.
	client.Timeout = 10 * time.Second

	data, err := downloadSearchTorrent(context.Background(), client, redirectSrv.URL)
	if err != nil {
		t.Fatalf("unexpected error for HTTPS→HTTPS redirect: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data from redirect chain")
	}
}

// =========================================================================
// newDownloadClient — CheckRedirect for redirect limit boundary
// =========================================================================

func TestNewDownloadClient_RedirectBoundary(t *testing.T) {
	client := newDownloadClient()
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}

	makeVia := func(n int) []*http.Request {
		via := make([]*http.Request, n)
		for i := range via {
			via[i] = req
		}
		return via
	}

	// 5 prior redirects — should be rejected (this is the 6th).
	if err := client.CheckRedirect(req, makeVia(5)); err == nil {
		t.Error("expected error for 5 prior redirects, got nil")
	}

	// 4 prior redirects — should be allowed (this is the 5th).
	if err := client.CheckRedirect(req, makeVia(4)); err != nil {
		t.Errorf("expected nil for 4 prior redirects, got: %v", err)
	}
}

func TestNewDownloadClient_ViaEmpty(t *testing.T) {
	client := newDownloadClient()
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	err := client.CheckRedirect(req, []*http.Request{})
	if err != nil {
		t.Errorf("expected nil for empty via, got: %v", err)
	}
}

// =========================================================================
// downloadSearchTorrent — CheckRedirect boundary (via length rejection)
// =========================================================================

func TestDownloadSearchTorrent_TooManyRedirects_FiveExact(t *testing.T) {
	// Exactly 5 redirects before final data. The redirect limit is 5
	// (len(via) >= 5 rejects). The 6th request attempt is the 302 response
	// from the 5th redirect, which is what CheckRedirect sees as via=5.
	// If the boundary is widened to >5, the redirect follows and we get data.
	var redirectCount int
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount <= 5 {
			http.Redirect(w, r, srv.URL, http.StatusFound)
		} else {
			w.Header().Set("Content-Type", "application/x-bittorrent")
			_, _ = w.Write([]byte("d8:announce..."))
		}
	}))
	defer srv.Close()

	client := newDownloadClient()
	// downloadSearchTorrent overrides CheckRedirect with its own, same limit.
	_, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err == nil {
		// If this succeeds, CheckRedirect allowed 5 prior redirects
		// meaning the boundary was widened. That's wrong.
		t.Error("expected error for 5 redirects, got nil (boundary widened?)")
	}
}

// =========================================================================
// downloadSearchTorrent — CheckRedirect via[0] scheme test
// =========================================================================

func TestDownloadSearchTorrent_HttpToHttpRedirect(t *testing.T) {
	// HTTP→HTTP redirect: via[0].Scheme == "http", so the downgrade check
	// should NOT trigger (it checks for https→http).
	var redirectCount int
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount == 1 {
			http.Redirect(w, r, srv.URL+"/final", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write([]byte("d8:announce..."))
	}))
	defer srv.Close()

	client := srv.Client()
	data, err := downloadSearchTorrent(context.Background(), client, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error for HTTP→HTTP redirect: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data from redirect chain")
	}
}
