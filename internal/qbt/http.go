package qbt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"
)

// HTTPClient is a qBittorrent API client that communicates over HTTP.
// It handles SID cookie-based authentication and transparently re-authenticates
// when a 403 response is received.
type HTTPClient struct {
	baseURL  string
	username string
	password string

	mu  sync.Mutex
	sid string // current session cookie value

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
// It is safe to call concurrently; the mutex serialises authentication attempts.
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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("qbt login: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK || string(body) == "Fails." {
		return fmt.Errorf("qbt login: authentication failed (status %d, body: %s)", resp.StatusCode, body)
	}

	// Extract SID from Set-Cookie header.
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "SID" {
			c.sid = cookie.Value
			return nil
		}
	}
	return fmt.Errorf("qbt login: SID cookie not found in response")
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
	c.mu.Unlock()
	attachCookie(req, sid)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusForbidden {
		return resp, nil
	}
	// Drain and close the 403 body before retrying.
	resp.Body.Close() //nolint:errcheck

	// Re-authenticate under lock to prevent multiple simultaneous logins.
	c.mu.Lock()
	loginErr := c.loginLocked(ctx)
	newSID := c.sid
	c.mu.Unlock()

	if loginErr != nil {
		return nil, fmt.Errorf("qbt re-auth: %w", loginErr)
	}

	retryReq, err := buildReq()
	if err != nil {
		return nil, fmt.Errorf("qbt rebuild request after re-auth: %w", err)
	}
	attachCookie(retryReq, newSID)

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

// attachCookie sets the SID session cookie on req, replacing any existing SID.
func attachCookie(req *http.Request, sid string) {
	// Remove any existing SID cookies to avoid duplicate/stale values, then add
	// the current one. http.Request.Header stores cookies under "Cookie".
	existing := req.Cookies()
	req.Header.Del("Cookie")
	for _, c := range existing {
		if c.Name != "SID" {
			req.AddCookie(c)
		}
	}
	req.AddCookie(&http.Cookie{Name: "SID", Value: sid})
}

// AddMagnet adds a torrent by magnet URI and assigns it to category.
func (c *HTTPClient) AddMagnet(ctx context.Context, magnet string, category string) error {
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qbt list torrents: unexpected status %d", resp.StatusCode)
	}

	var torrents []Torrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, fmt.Errorf("qbt list torrents: decode response: %w", err)
	}
	return torrents, nil
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
	defer resp.Body.Close()

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
