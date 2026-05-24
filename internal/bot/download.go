package bot

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// isPublicHostname checks whether host is a public (internet-routable) address.
//
// It first parses host as an IP literal. If it is a valid IP, it checks for
// private, loopback, link-local unicast, and unspecified ranges.
// Non-IP hostnames (e.g. "example.com") are allowed without DNS resolution
// for now; only obviously invalid inputs (empty, contains spaces) are rejected.
func isPublicHostname(host string) error {
	ip := net.ParseIP(host)
	if ip != nil {
		switch {
		case ip.IsLoopback():
			return fmt.Errorf("host %q is loopback", host)
		case ip.IsPrivate():
			return fmt.Errorf("host %q is private", host)
		case ip.IsLinkLocalUnicast():
			return fmt.Errorf("host %q is link-local unicast", host)
		case ip.IsUnspecified():
			return fmt.Errorf("host %q is unspecified", host)
		}
	}

	// Not a valid IP literal — could be a hostname or invalid input.
	if host == "" || strings.Contains(host, " ") {
		return fmt.Errorf("invalid hostname %q", host)
	}

	// TODO: add DNS resolution for hostnames.
	return nil
}

// newDownloadClient returns an *http.Client with a 15-second timeout and a
// redirect policy that limits redirects to 5 and rejects HTTPS→HTTP downgrades.
func newDownloadClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			if len(via) > 0 && via[0].URL.Scheme == "https" && req.URL.Scheme == "http" {
				return fmt.Errorf("https to http downgrade")
			}
			return nil
		},
	}
}

// downloadFile fetches raw bytes from url with safety checks:
//   - only http/https schemes are accepted
//   - if checkSSRF is true, validates hostname via ssrfCheck
//   - response must be 200 OK
//   - Content-Type must not be text/html
//   - body is limited to 10 MB
func downloadFile(ctx context.Context, client *http.Client, urlStr string, checkSSRF bool) ([]byte, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	if checkSSRF {
		if err := ssrfCheck(u.Hostname()); err != nil {
			return nil, fmt.Errorf("unsafe hostname: %w", err)
		}
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)

	// Install redirect policy so callers that pass a bare client (e.g.
	// httpsSrv.Client()) still get redirect safety.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("stopped after 5 redirects")
		}
		if via[0].URL.Scheme == "https" && req.URL.Scheme == "http" {
			return fmt.Errorf("https to http downgrade")
		}
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(ct), "text/html") {
		return nil, fmt.Errorf("unexpected content-type %q", ct)
	}

	const maxSize = 10 * 1024 * 1024 // 10 MB
	body := io.LimitReader(resp.Body, maxSize+1)
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if len(data) > maxSize {
		return nil, fmt.Errorf("response too large")
	}

	return data, nil
}

// downloadSearchTorrent calls downloadFile with SSRF protection enabled.
// Use for automated/untrusted download paths.
func downloadSearchTorrent(ctx context.Context, client *http.Client, urlStr string) ([]byte, error) {
	return downloadFile(ctx, client, urlStr, true)
}

// downloadUserTorrent calls downloadFile without SSRF protection.
// Use for user-initiated downloads from search results, where the URL came
// from qBittorrent's trusted search plugins and the user explicitly chose it.
func downloadUserTorrent(ctx context.Context, client *http.Client, urlStr string) ([]byte, error) {
	return downloadFile(ctx, client, urlStr, false)
}

// downloadUserTorrentFn is a package-level variable for test injection.
var downloadUserTorrentFn = downloadUserTorrent

// ssrfCheck is a package-level variable for test injection. Tests that use
// httptest servers on loopback can set it to nil.
var ssrfCheck = isPublicHostname
