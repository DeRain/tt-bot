package qbt

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// --- helpers ----------------------------------------------------------------

func newTestServer(t *testing.T, handler http.Handler) (*httptest.Server, *HTTPClient) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewHTTPClient(srv.URL, "admin", "testpass")
	return srv, client
}

// loginHandler returns an http.HandlerFunc that simulates a successful login by
// setting a SID cookie.
func loginHandler(sid string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: sid})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ok."))
	}
}

// parseMultipart parses the multipart body from r and returns the form values
// and any file parts keyed by field name.
func parseMultipart(t *testing.T, r *http.Request) (fields map[string]string, files map[string][]byte) {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		t.Fatalf("expected multipart content-type, got %q: %v", r.Header.Get("Content-Type"), err)
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	fields = make(map[string]string)
	files = make(map[string][]byte)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("reading multipart: %v", err)
		}
		data, _ := io.ReadAll(part)
		if part.FileName() != "" {
			files[part.FormName()] = data
		} else {
			fields[part.FormName()] = string(data)
		}
	}
	return fields, files
}

// --- Login tests ------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	const wantSID = "abc123"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(wantSID))

	_, client := newTestServer(t, mux)

	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if client.sid != wantSID {
		t.Errorf("sid = %q, want %q", client.sid, wantSID)
	}
}

func TestLogin_FailsBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Fails."))
	})

	_, client := newTestServer(t, mux)

	err := client.Login(context.Background())
	if err == nil {
		t.Fatal("expected error for Fails. body, got nil")
	}
}

func TestLogin_Fails403(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	_, client := newTestServer(t, mux)

	err := client.Login(context.Background())
	if err == nil {
		t.Fatal("expected error for 403 response, got nil")
	}
}

// TestLogin_Success204AuthBypass verifies that a 204 No Content response
// (qBittorrent v5.1+ auth-bypass mode) is treated as a successful login and
// that no SID is stored (server identifies client by IP).
func TestLogin_Success204AuthBypass(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		// No Set-Cookie header — auth bypass grants access by IP.
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := newTestServer(t, mux)

	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login() error = %v, want nil for 204 bypass response", err)
	}
	if client.sid != "" {
		t.Errorf("sid = %q, want empty string for auth-bypass mode", client.sid)
	}
}

// TestLogin_Success200NoCookie verifies that a 200 OK with body "Ok." but
// without a Set-Cookie header is treated as a successful login (sid stays "").
func TestLogin_Success200NoCookie(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		// 200 OK with success body but no SID cookie.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ok."))
	})

	_, client := newTestServer(t, mux)

	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login() error = %v, want nil for 200+Ok. without cookie", err)
	}
	if client.sid != "" {
		t.Errorf("sid = %q, want empty string when no SID cookie set", client.sid)
	}
}

// TestLogin_Success204WithQBTCookie verifies that a 204 response with the
// qBittorrent v5.2+ port-specific cookie (QBT_SID_<port>) is accepted and
// the cookie value is stored for subsequent requests.
func TestLogin_Success204WithQBTCookie(t *testing.T) {
	const wantSID = "newstyle-cookie-value"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "QBT_SID_8080", Value: wantSID})
		w.WriteHeader(http.StatusNoContent)
	})

	_, client := newTestServer(t, mux)

	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login() error = %v, want nil for 204 with QBT_SID cookie", err)
	}
	if client.sid != wantSID {
		t.Errorf("sid = %q, want %q", client.sid, wantSID)
	}
	if client.sessionName != "QBT_SID_8080" {
		t.Errorf("sessionName = %q, want %q", client.sessionName, "QBT_SID_8080")
	}
}

// TestAuthBypass_NoSIDCookieSent verifies the end-to-end auth-bypass flow:
// after a 204 login (no SID cookie), follow-up requests do not include a
// Cookie header containing "SID=".
func TestAuthBypass_NoSIDCookieSent(t *testing.T) {
	var capturedCookieHeader string

	mux := http.NewServeMux()
	// Login returns 204 — bypass mode, no cookie.
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	// Capture the Cookie header on the follow-up request.
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		capturedCookieHeader = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	})

	_, client := newTestServer(t, mux)

	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	_, err := client.ListTorrents(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}

	// In bypass mode no SID cookie should appear in the request.
	if strings.Contains(capturedCookieHeader, "SID=") {
		t.Errorf("Cookie header = %q, must not contain SID= in auth-bypass mode", capturedCookieHeader)
	}
}

// --- AddMagnet tests --------------------------------------------------------

func TestAddMagnet_SendsCorrectForm(t *testing.T) {
	const (
		wantMagnet   = "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0"
		wantCategory = "movies"
		sid          = "sid1"
	)

	var gotFields map[string]string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		gotFields, _ = parseMultipart(t, r)
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	if err := client.Login(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := client.AddMagnet(context.Background(), wantMagnet, wantCategory); err != nil {
		t.Fatalf("AddMagnet() error = %v", err)
	}

	if gotFields["urls"] != wantMagnet {
		t.Errorf("urls = %q, want %q", gotFields["urls"], wantMagnet)
	}
	if gotFields["category"] != wantCategory {
		t.Errorf("category = %q, want %q", gotFields["category"], wantCategory)
	}
}

func TestAddMagnet_NoCategory(t *testing.T) {
	const sid = "sid1"
	var gotFields map[string]string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		gotFields, _ = parseMultipart(t, r)
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	if err := client.AddMagnet(context.Background(), "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0", ""); err != nil {
		t.Fatalf("AddMagnet() error = %v", err)
	}
	if _, ok := gotFields["category"]; ok {
		t.Errorf("category field should be absent when category is empty")
	}
}

// --- AddTorrentFile tests ---------------------------------------------------

func TestAddTorrentFile_SendsFileData(t *testing.T) {
	const (
		filename = "test.torrent"
		sid      = "sid2"
		category = "tv"
	)
	fileContent := []byte("fake torrent data")

	var gotFiles map[string][]byte
	var gotFields map[string]string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		gotFields, gotFiles = parseMultipart(t, r)
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.AddTorrentFile(context.Background(), filename, strings.NewReader(string(fileContent)), category)
	if err != nil {
		t.Fatalf("AddTorrentFile() error = %v", err)
	}

	if string(gotFiles["torrents"]) != string(fileContent) {
		t.Errorf("file data = %q, want %q", gotFiles["torrents"], fileContent)
	}
	if gotFields["category"] != category {
		t.Errorf("category = %q, want %q", gotFields["category"], category)
	}
}

// TestAddMagnet_DuplicateReturns409 verifies that a 409 Conflict response
// (qBittorrent v5.2+ duplicate-torrent signal) is treated as a successful no-op.
func TestAddMagnet_DuplicateReturns409(t *testing.T) {
	const sid = "sid-dup-mag"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	if err := client.AddMagnet(context.Background(), "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0", "movies"); err != nil {
		t.Fatalf("AddMagnet() error = %v, want nil for 409 duplicate", err)
	}
}

// TestAddTorrentFile_DuplicateReturns409 verifies that a 409 Conflict response
// (qBittorrent v5.2+ duplicate-torrent signal) is treated as a successful no-op.
func TestAddTorrentFile_DuplicateReturns409(t *testing.T) {
	const sid = "sid-dup-file"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.AddTorrentFile(context.Background(), "dup.torrent", strings.NewReader("fake"), "tv")
	if err != nil {
		t.Fatalf("AddTorrentFile() error = %v, want nil for 409 duplicate", err)
	}
}

// TestAddMagnet_500StillFails is a regression guard that ensures AddMagnet
// still returns an error for server-side failures (500 Internal Server Error).
func TestAddMagnet_500StillFails(t *testing.T) {
	const sid = "sid-mag-500"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.AddMagnet(context.Background(), "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0", "")
	if err == nil {
		t.Fatal("AddMagnet() error = nil, want error for 500 response")
	}
}

// --- ListTorrents tests -----------------------------------------------------

func TestListTorrents_ParsesResponse(t *testing.T) {
	sid := "sid3"
	want := []Torrent{
		{Hash: "aabbcc", Name: "Ubuntu", State: "downloading", Progress: 0.5, Size: 1024},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	got, err := client.ListTorrents(context.Background(), ListOptions{Filter: FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(got) != 1 || got[0].Hash != want[0].Hash {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestListTorrents_FilterAndPagination(t *testing.T) {
	sid := "sid4"
	var gotQuery url.Values

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	opts := ListOptions{Filter: FilterActive, Limit: 5, Offset: 10}
	_, err := client.ListTorrents(context.Background(), opts)
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}

	if gotQuery.Get("filter") != "active" {
		t.Errorf("filter = %q, want %q", gotQuery.Get("filter"), "active")
	}
	if gotQuery.Get("limit") != "5" {
		t.Errorf("limit = %q, want %q", gotQuery.Get("limit"), "5")
	}
	if gotQuery.Get("offset") != "10" {
		t.Errorf("offset = %q, want %q", gotQuery.Get("offset"), "10")
	}
}

// --- Categories tests -------------------------------------------------------

func TestCategories_ReturnsSortedSlice(t *testing.T) {
	sid := "sid5"
	apiResponse := map[string]Category{
		"tv":     {Name: "tv", SavePath: "/data/tv"},
		"movies": {Name: "movies", SavePath: "/data/movies"},
		"anime":  {Name: "anime", SavePath: "/data/anime"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/categories", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apiResponse)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	got, err := client.Categories(context.Background())
	if err != nil {
		t.Fatalf("Categories() error = %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// Verify sorted order.
	wantNames := []string{"anime", "movies", "tv"}
	for i, cat := range got {
		if cat.Name != wantNames[i] {
			t.Errorf("got[%d].Name = %q, want %q", i, cat.Name, wantNames[i])
		}
	}
}

// --- PauseTorrents tests ----------------------------------------------------

func TestPauseTorrents_SendsCorrectForm(t *testing.T) {
	const sid = "sid-pause"
	var gotBody string
	var gotMethod string
	var gotPath string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/stop", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	hashes := []string{"abc123", "def456"}
	if err := client.PauseTorrents(context.Background(), hashes); err != nil {
		t.Fatalf("PauseTorrents() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v2/torrents/stop" {
		t.Errorf("path = %q, want /api/v2/torrents/stop", gotPath)
	}

	wantBody := "hashes=abc123%7Cdef456"
	if gotBody != wantBody {
		t.Errorf("body = %q, want %q", gotBody, wantBody)
	}
}

func TestPauseTorrents_ReauthOn403(t *testing.T) {
	const newSID = "new-sid-pause"
	loginCount := 0
	callCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		loginCount++
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: newSID})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ok."))
	})
	mux.HandleFunc("/api/v2/torrents/stop", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		cookie, err := r.Cookie("SID")
		if err != nil || cookie.Value != newSID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())
	// Invalidate SID to trigger re-auth.
	client.mu.Lock()
	client.sid = "stale"
	client.mu.Unlock()

	if err := client.PauseTorrents(context.Background(), []string{"hash1"}); err != nil {
		t.Fatalf("PauseTorrents() error after re-auth = %v", err)
	}
	if loginCount < 2 {
		t.Errorf("loginCount = %d, want >= 2", loginCount)
	}
}

// --- ResumeTorrents tests ---------------------------------------------------

func TestResumeTorrents_SendsCorrectForm(t *testing.T) {
	const sid = "sid-resume"
	var gotBody string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/start", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	if err := client.ResumeTorrents(context.Background(), []string{"xyz789"}); err != nil {
		t.Fatalf("ResumeTorrents() error = %v", err)
	}

	wantBody := "hashes=xyz789"
	if gotBody != wantBody {
		t.Errorf("body = %q, want %q", gotBody, wantBody)
	}
}

func TestPauseTorrents_ErrorOnNon200(t *testing.T) {
	const sid = "sid-pause-err"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/stop", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.PauseTorrents(context.Background(), []string{"hash1"})
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestResumeTorrents_ErrorOnNon200(t *testing.T) {
	const sid = "sid-resume-err"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/start", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.ResumeTorrents(context.Background(), []string{"hash1"})
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

// --- DeleteTorrents tests (TEST-2: TASK-2) -----------------------------------

func TestDeleteTorrents_SendsCorrectForm_NoDeleteFiles(t *testing.T) {
	const sid = "sid-del"
	var gotBody string
	var gotMethod string
	var gotPath string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	hashes := []string{"abc123", "def456"}
	if err := client.DeleteTorrents(context.Background(), hashes, false); err != nil {
		t.Fatalf("DeleteTorrents() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v2/torrents/delete" {
		t.Errorf("path = %q, want /api/v2/torrents/delete", gotPath)
	}

	parsed, _ := url.ParseQuery(gotBody)
	if got := parsed.Get("hashes"); got != "abc123|def456" {
		t.Errorf("hashes = %q, want %q", got, "abc123|def456")
	}
	if got := parsed.Get("deleteFiles"); got != "false" {
		t.Errorf("deleteFiles = %q, want %q", got, "false")
	}
}

func TestDeleteTorrents_SendsCorrectForm_WithDeleteFiles(t *testing.T) {
	const sid = "sid-del-files"
	var gotBody string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	if err := client.DeleteTorrents(context.Background(), []string{"xyz789"}, true); err != nil {
		t.Fatalf("DeleteTorrents() error = %v", err)
	}

	parsed, _ := url.ParseQuery(gotBody)
	if got := parsed.Get("deleteFiles"); got != "true" {
		t.Errorf("deleteFiles = %q, want %q", got, "true")
	}
	if got := parsed.Get("hashes"); got != "xyz789" {
		t.Errorf("hashes = %q, want %q", got, "xyz789")
	}
}

func TestDeleteTorrents_ErrorOnNon200(t *testing.T) {
	const sid = "sid-del-err"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.DeleteTorrents(context.Background(), []string{"hash1"}, false)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

// --- Auto re-auth tests -----------------------------------------------------

func TestAutoReauth_On403(t *testing.T) {
	const (
		initialSID = "old-sid"
		newSID     = "new-sid"
	)

	loginCount := 0
	torrentCallCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		loginCount++
		// First login sets initialSID; re-auth sets newSID.
		sid := initialSID
		if loginCount > 1 {
			sid = newSID
		}
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: sid})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ok."))
	})
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		torrentCallCount++
		cookie, err := r.Cookie("SID")
		if err != nil || cookie.Value != newSID {
			// Return 403 on first attempt (old or missing SID).
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// Succeed on retry with newSID.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Torrent{{Hash: "hash1", Name: "Test"}})
	})

	_, client := newTestServer(t, mux)

	// Perform initial login to get initialSID.
	if err := client.Login(context.Background()); err != nil {
		t.Fatal(err)
	}

	got, err := client.ListTorrents(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}

	if len(got) != 1 || got[0].Hash != "hash1" {
		t.Errorf("unexpected result after re-auth: %+v", got)
	}
	if loginCount != 2 {
		t.Errorf("loginCount = %d, want 2 (initial + re-auth)", loginCount)
	}
	if torrentCallCount != 2 {
		t.Errorf("torrentCallCount = %d, want 2 (403 + retry)", torrentCallCount)
	}
}

// --- ListFiles tests (TEST-2, TASK-3) ----------------------------------------

// TestListFiles_ParsesResponse verifies that ListFiles correctly deserialises
// the JSON array returned by /api/v2/torrents/files.
func TestListFiles_ParsesResponse(t *testing.T) {
	const (
		sid  = "sid-lf"
		hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	)
	want := []TorrentFile{
		{Index: 0, Name: "Season 1/ep01.mkv", Size: 1073741824, Progress: 0.5, Priority: FilePriorityNormal},
		{Index: 1, Name: "Season 1/ep02.mkv", Size: 734003200, Progress: 0.0, Priority: FilePrioritySkip},
	}

	var gotQuery url.Values
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/files", func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	got, err := client.ListFiles(context.Background(), hash)
	if err != nil {
		t.Fatalf("ListFiles() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != want[0].Name || got[0].Priority != want[0].Priority {
		t.Errorf("got[0] = %+v, want %+v", got[0], want[0])
	}
	if got[1].Priority != FilePrioritySkip {
		t.Errorf("got[1].Priority = %d, want %d (Skip)", got[1].Priority, FilePrioritySkip)
	}
	if gotQuery.Get("hash") != hash {
		t.Errorf("hash query param = %q, want %q", gotQuery.Get("hash"), hash)
	}
}

// TestListFiles_ErrorOnNon200 verifies that ListFiles returns a non-nil error
// when the server responds with a non-200 status.
func TestListFiles_ErrorOnNon200(t *testing.T) {
	const sid = "sid-lf-err"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/files", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	_, err := client.ListFiles(context.Background(), "somehash")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

// --- SetFilePriority tests (TEST-2, TASK-3) -----------------------------------

// TestSetFilePriority_SendsCorrectForm verifies that SetFilePriority sends a
// POST to /api/v2/torrents/filePrio with the correct hash, id, and priority fields.
func TestSetFilePriority_SendsCorrectForm(t *testing.T) {
	const (
		sid  = "sid-sfp"
		hash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	var gotBody string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/filePrio", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.SetFilePriority(context.Background(), hash, []int{0, 2}, FilePriorityHigh)
	if err != nil {
		t.Fatalf("SetFilePriority() error = %v", err)
	}

	parsed, _ := url.ParseQuery(gotBody)
	if got := parsed.Get("hash"); got != hash {
		t.Errorf("hash = %q, want %q", got, hash)
	}
	if got := parsed.Get("id"); got != "0|2" {
		t.Errorf("id = %q, want %q", got, "0|2")
	}
	if got := parsed.Get("priority"); got != "6" {
		t.Errorf("priority = %q, want %q", got, "6")
	}
}

// TestSetFilePriority_ErrorOnNon200 verifies that SetFilePriority returns a
// non-nil error when the server responds with a non-200 status.
func TestSetFilePriority_ErrorOnNon200(t *testing.T) {
	const sid = "sid-sfp-err"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler(sid))
	mux.HandleFunc("/api/v2/torrents/filePrio", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.SetFilePriority(context.Background(), "hash", []int{0}, FilePrioritySkip)
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

// --- validateMagnetURI tests ------------------------------------------------

// TestValidateMagnetURI exercises the client-side magnet URI validator.
func TestValidateMagnetURI(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid basic",
			input:   "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0",
			wantErr: false,
		},
		{
			name:    "valid with extra params",
			input:   "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=foo&tr=udp://t/",
			wantErr: false,
		},
		{
			name:    "valid uppercase hash",
			input:   "magnet:?xt=urn:btih:3B245504CF5F11BBDBE1201CEA6A6BF45AEE1BC0",
			wantErr: false,
		},
		{
			name:    "invalid empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid not a magnet",
			input:   "not-a-magnet",
			wantErr: true,
		},
		{
			name:    "invalid http scheme",
			input:   "http://example.com/",
			wantErr: true,
		},
		{
			name:    "invalid missing xt",
			input:   "magnet:?dn=foo",
			wantErr: true,
		},
		{
			name:    "invalid wrong urn type",
			input:   "magnet:?xt=urn:sha1:abc",
			wantErr: true,
		},
		{
			name:    "invalid hash too short (39 chars)",
			input:   "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc",
			wantErr: true,
		},
		{
			name:    "invalid hash too long v2 (64 chars)",
			input:   "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc03b245504cf5f11bbdbe12012",
			wantErr: true,
		},
		{
			name:    "invalid non-hex chars in hash",
			input:   "magnet:?xt=urn:btih:gggggggggggggggggggggggggggggggggggggggg",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMagnetURI(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMagnetURI(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestAddMagnet_RejectsInvalidInput verifies that AddMagnet returns an error
// for an invalid magnet URI and makes NO HTTP request to qBittorrent.
func TestAddMagnet_RejectsInvalidInput(t *testing.T) {
	handlerCalled := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", loginHandler("sid-reject"))
	mux.HandleFunc("/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled++
		t.Errorf("HTTP handler should not have been called for invalid magnet input")
		w.WriteHeader(http.StatusOK)
	})

	_, client := newTestServer(t, mux)
	_ = client.Login(context.Background())

	err := client.AddMagnet(context.Background(), "not-a-magnet", "movies")
	if err == nil {
		t.Fatal("AddMagnet() error = nil, want error for invalid magnet URI")
	}
	if handlerCalled != 0 {
		t.Errorf("HTTP handler called %d time(s), want 0", handlerCalled)
	}
}
