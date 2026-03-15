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

// --- AddMagnet tests --------------------------------------------------------

func TestAddMagnet_SendsCorrectForm(t *testing.T) {
	const (
		wantMagnet   = "magnet:?xt=urn:btih:abc"
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

	if err := client.AddMagnet(context.Background(), "magnet:?xt=urn:btih:xyz", ""); err != nil {
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
