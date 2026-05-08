package qbt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// btihV1RE matches a v1 BitTorrent info-hash in a urn:btih:<HASH> xt value.
// A v1 hash is exactly 40 hexadecimal characters (SHA-1). v2 (64-char SHA-256)
// is not supported by this bot yet (see docs/features/torrent-control/decisions.md).
var btihV1RE = regexp.MustCompile(`(?i)^urn:btih:([0-9a-f]{40})$`)

// validateMagnetURI returns a non-nil error if magnet is not a valid magnet URI
// containing at least one v1 BitTorrent info-hash (urn:btih:<40-hex-char hash>).
// This pre-validates user input before sending to qBittorrent so that a 409
// Conflict response from /torrents/add can be unambiguously interpreted as
// "duplicate torrent" rather than "malformed input".
func validateMagnetURI(magnet string) error {
	if !strings.HasPrefix(magnet, "magnet:?") {
		return errors.New("invalid magnet URI: missing magnet:? scheme")
	}

	// Parse everything after "magnet:?" as a query string.
	params, err := url.ParseQuery(magnet[len("magnet:?"):])
	if err != nil {
		return fmt.Errorf("invalid magnet URI: unparseable query parameters: %w", err)
	}

	xtValues, ok := params["xt"]
	if !ok || len(xtValues) == 0 {
		return errors.New("invalid magnet URI: missing xt parameter")
	}

	// At least one xt value must be a valid v1 btih hash.
	for _, xt := range xtValues {
		if btihV1RE.MatchString(xt) {
			return nil
		}
	}

	// Distinguish between wrong URN type and wrong hash length/chars.
	for _, xt := range xtValues {
		if strings.HasPrefix(strings.ToLower(xt), "urn:btih:") {
			hash := xt[len("urn:btih:"):]
			if len(hash) != 40 {
				return fmt.Errorf("invalid magnet URI: info-hash must be 40 hex characters (got %d)", len(hash))
			}
			return errors.New("invalid magnet URI: info-hash contains non-hex characters")
		}
	}
	return errors.New("invalid magnet URI: xt value not a v1 BitTorrent info-hash")
}

// HTTPClient is a qBittorrent API client that communicates over HTTP.
// It handles SID cookie-based authentication and transparently re-authenticates
// when a 403 response is received.
type HTTPClient struct {
	baseURL  string
	username string
	password string

	mu          sync.Mutex
	sid         string // current session cookie value
	sessionName string // cookie name: "SID" (pre-v5.2) or "QBT_SID_<port>" (v5.2+)

	httpClient *http.Client
}

// NewHTTPClient constructs a new HTTPClient with the given base URL and credentials.
// baseURL should not have a trailing slash, e.g. "http://localhost:8080".
func NewHTTPClient(baseURL, username, password string) *HTTPClient {
	return &HTTPClient{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Login authenticates with qBittorrent and stores the SID session cookie.
// It is safe to call concurrently; the mutex serializes authentication attempts.
func (c *HTTPClient) Login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loginLocked(ctx)
}

// loginLocked performs the actual login request. Must be called with c.mu held.
func (c *HTTPClient) loginLocked(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v2/auth/login",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("qbt login: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qbt login: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("qbt login: read body: %w", err)
	}

	// Accept any 2xx status. qBittorrent v5.1+ returns 204 No Content when the
	// client IP is in the auth-bypass subnet whitelist; in that case no SID
	// cookie is issued and the client is identified by IP alone.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qbt login: authentication failed (status %d, body: %s)", resp.StatusCode, body)
	}
	// "Fails." is qBittorrent's explicit failure signal on 200 responses.
	if resp.StatusCode == http.StatusOK && string(body) == "Fails." {
		return fmt.Errorf("qbt login: authentication failed (status %d, body: %s)", resp.StatusCode, body)
	}

	// Extract the session cookie from Set-Cookie headers. qBittorrent uses:
	//   - "SID" in pre-v5.2 releases
	//   - "QBT_SID_<port>" in v5.2+ releases
	// Accept any cookie whose name is "SID" or starts with "QBT_SID_".
	// If no matching cookie is found (auth-bypass mode), leave sid as "" —
	// attachCookie will not add an empty SID header and the server identifies
	// the client by IP.
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "SID" || strings.HasPrefix(cookie.Name, "QBT_SID_") {
			c.sid = cookie.Value
			c.sessionName = cookie.Name
			return nil
		}
	}
	// No session cookie — auth-bypass mode or server chose not to set one.
	// Clear any stale session info and proceed without cookie-based auth.
	c.sid = ""
	c.sessionName = ""
	return nil
}

// doWithRetry executes the request produced by buildReq, attaching the session
// cookie. If the server responds with 403 it re-authenticates once and retries
// by calling buildReq again to obtain a fresh request (necessary for multipart
// bodies that cannot be re-read). The caller is responsible for closing the
// returned response body.
func (c *HTTPClient) doWithRetry(ctx context.Context, buildReq func() (*http.Request, error)) (*http.Response, error) {
	req, err := buildReq()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	sid := c.sid
	sessionName := c.sessionName
	c.mu.Unlock()
	attachCookie(req, sessionName, sid)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusForbidden {
		return resp, nil
	}
	// Drain and close the 403 body before retrying.
	_ = resp.Body.Close()

	// Re-authenticate under lock to prevent multiple simultaneous logins.
	c.mu.Lock()
	loginErr := c.loginLocked(ctx)
	newSID := c.sid
	newSessionName := c.sessionName
	c.mu.Unlock()

	if loginErr != nil {
		return nil, fmt.Errorf("qbt re-auth: %w", loginErr)
	}

	retryReq, err := buildReq()
	if err != nil {
		return nil, fmt.Errorf("qbt rebuild request after re-auth: %w", err)
	}
	attachCookie(retryReq, newSessionName, newSID)

	retryResp, err := c.httpClient.Do(retryReq)
	if err != nil {
		return nil, fmt.Errorf("qbt retry after re-auth: %w", err)
	}
	return retryResp, nil
}

// doWithAuth executes req attaching the session cookie. If the server responds
// with 403 it re-authenticates once (holding the mutex to avoid thundering herd)
// and retries the request.
// The caller is responsible for closing the returned response body.
func (c *HTTPClient) doWithAuth(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	// Capture req in a closure so doWithRetry can clone it for the retry.
	// Body-less or pre-buffered requests are safe to clone.
	original := req
	return c.doWithRetry(ctx, func() (*http.Request, error) {
		return original.Clone(ctx), nil
	})
}

// attachCookie sets the session cookie on req, replacing any existing session
// cookie (either the legacy "SID" or the v5.2+ "QBT_SID_<port>" variant).
// When sid is empty (auth-bypass mode) no session cookie is added; stale
// cookies are still stripped so the server sees a clean request.
// name defaults to "SID" when empty for backwards compatibility.
func attachCookie(req *http.Request, name, sid string) {
	if name == "" {
		name = "SID"
	}
	// Remove any existing session cookies (both legacy "SID" and "QBT_SID_*")
	// to avoid duplicate/stale values. http.Request.Header stores cookies under "Cookie".
	existing := req.Cookies()
	req.Header.Del("Cookie")
	for _, c := range existing {
		if c.Name != "SID" && !strings.HasPrefix(c.Name, "QBT_SID_") {
			req.AddCookie(c)
		}
	}
	if sid != "" {
		req.AddCookie(&http.Cookie{Name: name, Value: sid})
	}
}

// AddMagnet adds a torrent by magnet URI and assigns it to category.
func (c *HTTPClient) AddMagnet(ctx context.Context, magnet string, category string) error {
	if err := validateMagnetURI(magnet); err != nil {
		return fmt.Errorf("qbt add magnet: %w", err)
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if err := mw.WriteField("urls", magnet); err != nil {
		return fmt.Errorf("qbt add magnet: write urls field: %w", err)
	}
	if category != "" {
		if err := mw.WriteField("category", category); err != nil {
			return fmt.Errorf("qbt add magnet: write category field: %w", err)
		}
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("qbt add magnet: close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v2/torrents/add", &buf)
	if err != nil {
		return fmt.Errorf("qbt add magnet: build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.doWithAuth(req)
	if err != nil {
		return fmt.Errorf("qbt add magnet: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// HTTP 409 Conflict means the torrent already exists (qBittorrent v5.2+).
	// Treat as a successful no-op — the torrent is already present.
	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbt add magnet: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// AddTorrentFile uploads a .torrent file and assigns it to category.
func (c *HTTPClient) AddTorrentFile(ctx context.Context, filename string, data io.Reader, category string) error {
	// Buffer file data so we can retry on 403 without re-reading the reader.
	fileBytes, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("qbt add torrent file: read data: %w", err)
	}

	buildRequest := func() (*http.Request, error) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)

		part, err := mw.CreateFormFile("torrents", filename)
		if err != nil {
			return nil, fmt.Errorf("create form file: %w", err)
		}
		if _, err := part.Write(fileBytes); err != nil {
			return nil, fmt.Errorf("write file bytes: %w", err)
		}

		if category != "" {
			if err := mw.WriteField("category", category); err != nil {
				return nil, fmt.Errorf("write category field: %w", err)
			}
		}
		if err := mw.Close(); err != nil {
			return nil, fmt.Errorf("close multipart writer: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			c.baseURL+"/api/v2/torrents/add", &buf)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", mw.FormDataContentType())
		return req, nil
	}

	resp, err := c.doWithRetry(ctx, buildRequest)
	if err != nil {
		return fmt.Errorf("qbt add torrent file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// HTTP 409 Conflict means the torrent already exists (qBittorrent v5.2+).
	// Treat as a successful no-op — the torrent is already present.
	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbt add torrent file: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ListTorrents returns torrents matching opts.
func (c *HTTPClient) ListTorrents(ctx context.Context, opts ListOptions) ([]Torrent, error) {
	u, err := url.Parse(c.baseURL + "/api/v2/torrents/info")
	if err != nil {
		return nil, fmt.Errorf("qbt list torrents: parse URL: %w", err)
	}

	q := u.Query()
	if opts.Filter != "" {
		q.Set("filter", string(opts.Filter))
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("qbt list torrents: build request: %w", err)
	}

	resp, err := c.doWithAuth(req)
	if err != nil {
		return nil, fmt.Errorf("qbt list torrents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qbt list torrents: unexpected status %d", resp.StatusCode)
	}

	var torrents []Torrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, fmt.Errorf("qbt list torrents: decode response: %w", err)
	}
	return torrents, nil
}

// PauseTorrents pauses (stops) the given torrents by info-hash.
// Hashes are sent as a pipe-separated string to /api/v2/torrents/stop.
// Note: qBittorrent v5+ renamed /pause to /stop.
func (c *HTTPClient) PauseTorrents(ctx context.Context, hashes []string) error {
	form := url.Values{}
	form.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v2/torrents/stop",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("qbt pause torrents: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithAuth(req)
	if err != nil {
		return fmt.Errorf("qbt pause torrents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbt pause torrents: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ResumeTorrents resumes (starts) the given torrents by info-hash.
// Hashes are sent as a pipe-separated string to /api/v2/torrents/start.
// Note: qBittorrent v5+ renamed /resume to /start.
func (c *HTTPClient) ResumeTorrents(ctx context.Context, hashes []string) error {
	form := url.Values{}
	form.Set("hashes", strings.Join(hashes, "|"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v2/torrents/start",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("qbt resume torrents: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithAuth(req)
	if err != nil {
		return fmt.Errorf("qbt resume torrents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbt resume torrents: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// DeleteTorrents removes one or more torrents by info-hash.
// If deleteFiles is true, the associated downloaded data is also removed from disk.
// Hashes are sent as a pipe-separated string to /api/v2/torrents/delete.
func (c *HTTPClient) DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error {
	form := url.Values{}
	form.Set("hashes", strings.Join(hashes, "|"))
	form.Set("deleteFiles", strconv.FormatBool(deleteFiles))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v2/torrents/delete",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("qbt delete torrents: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithAuth(req)
	if err != nil {
		return fmt.Errorf("qbt delete torrents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbt delete torrents: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ListFiles returns all files contained within the torrent identified by hash.
func (c *HTTPClient) ListFiles(ctx context.Context, hash string) ([]TorrentFile, error) {
	u, err := url.Parse(c.baseURL + "/api/v2/torrents/files")
	if err != nil {
		return nil, fmt.Errorf("qbt list files: parse URL: %w", err)
	}

	q := u.Query()
	q.Set("hash", hash)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("qbt list files: build request: %w", err)
	}

	resp, err := c.doWithAuth(req)
	if err != nil {
		return nil, fmt.Errorf("qbt list files: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qbt list files: unexpected status %d", resp.StatusCode)
	}

	var files []TorrentFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("qbt list files: decode response: %w", err)
	}
	return files, nil
}

// SetFilePriority sets the download priority for the given file indices within
// the torrent identified by hash.
func (c *HTTPClient) SetFilePriority(ctx context.Context, hash string, fileIndices []int, priority FilePriority) error {
	indices := make([]string, len(fileIndices))
	for i, idx := range fileIndices {
		indices[i] = strconv.Itoa(idx)
	}

	form := url.Values{}
	form.Set("hash", hash)
	form.Set("id", strings.Join(indices, "|"))
	form.Set("priority", strconv.Itoa(int(priority)))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v2/torrents/filePrio",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("qbt set file priority: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.doWithAuth(req)
	if err != nil {
		return fmt.Errorf("qbt set file priority: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbt set file priority: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// Categories returns all configured categories sorted by name.
func (c *HTTPClient) Categories(ctx context.Context) ([]Category, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/v2/torrents/categories", nil)
	if err != nil {
		return nil, fmt.Errorf("qbt categories: build request: %w", err)
	}

	resp, err := c.doWithAuth(req)
	if err != nil {
		return nil, fmt.Errorf("qbt categories: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qbt categories: unexpected status %d", resp.StatusCode)
	}

	// The API returns map[string]Category.
	var raw map[string]Category
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("qbt categories: decode response: %w", err)
	}

	result := make([]Category, 0, len(raw))
	for _, cat := range raw {
		result = append(result, cat)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}
