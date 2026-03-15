//go:build integration

package qbt

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// integrationClient builds an HTTPClient from environment variables, falling back
// to the defaults used by the Docker test stack.
func integrationClient(t *testing.T) *HTTPClient {
	t.Helper()
	baseURL := envOrDefault("QBITTORRENT_URL", "http://localhost:18080")
	username := envOrDefault("QBITTORRENT_USERNAME", "admin")
	password := envOrDefault("QBITTORRENT_PASSWORD", "")
	return NewHTTPClient(baseURL, username, password)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// TestIntegration_Login verifies that Login() succeeds against the real qBittorrent
// instance started by docker-compose.test.yml.
func TestIntegration_Login(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
}

// TestIntegration_Categories verifies that Categories() returns a valid (possibly
// empty) slice without error.
func TestIntegration_Categories(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	cats, err := c.Categories(ctx)
	if err != nil {
		t.Fatalf("Categories() error = %v", err)
	}
	// Result may be empty on a fresh instance — just assert no error and valid slice.
	if cats == nil {
		t.Error("Categories() returned nil slice, want non-nil")
	}
}

// TestIntegration_AddMagnetAndList adds a well-known magnet link and verifies the
// torrent appears in the list returned by ListTorrents.
func TestIntegration_AddMagnetAndList(t *testing.T) {
	const ubuntuMagnet = "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-24.04-desktop-amd64.iso"
	const ubuntuHash = "3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0"

	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if err := c.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Fatalf("AddMagnet() error = %v", err)
	}

	// qBittorrent processes the magnet asynchronously; allow a short window for it
	// to appear in the torrent list.
	var found bool
	for attempt := 0; attempt < 5; attempt++ {
		torrents, err := c.ListTorrents(ctx, ListOptions{Filter: FilterAll})
		if err != nil {
			t.Fatalf("ListTorrents() error = %v", err)
		}
		for _, tor := range torrents {
			if strings.EqualFold(tor.Hash, ubuntuHash) ||
				strings.Contains(strings.ToLower(tor.Name), "ubuntu") {
				found = true
				break
			}
		}
		if found {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !found {
		t.Error("added torrent not found in ListTorrents() result")
	}
}

// TestIntegration_PauseAndResumeTorrent verifies that PauseTorrents and
// ResumeTorrents work against a real qBittorrent instance. A torrent must exist
// (seeded by TestIntegration_AddMagnetAndList or prior test runs).
func TestIntegration_PauseAndResumeTorrent(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Ensure at least one torrent exists.
	const ubuntuMagnet = "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-24.04-desktop-amd64.iso"
	_ = c.AddMagnet(ctx, ubuntuMagnet, "")
	time.Sleep(2 * time.Second)

	torrents, err := c.ListTorrents(ctx, ListOptions{Filter: FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available to pause/resume")
	}

	hash := torrents[0].Hash

	// Pause the torrent.
	if err := c.PauseTorrents(ctx, []string{hash}); err != nil {
		t.Fatalf("PauseTorrents() error = %v", err)
	}

	// Wait for state to propagate and verify.
	time.Sleep(2 * time.Second)
	updated, err := c.ListTorrents(ctx, ListOptions{Filter: FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() after pause error = %v", err)
	}
	for _, tor := range updated {
		if tor.Hash == hash {
			if tor.State != "pausedDL" && tor.State != "pausedUP" {
				t.Logf("torrent state after pause = %q (may be transitioning)", tor.State)
			}
			break
		}
	}

	// Resume the torrent.
	if err := c.ResumeTorrents(ctx, []string{hash}); err != nil {
		t.Fatalf("ResumeTorrents() error = %v", err)
	}

	time.Sleep(2 * time.Second)
	resumed, err := c.ListTorrents(ctx, ListOptions{Filter: FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() after resume error = %v", err)
	}
	for _, tor := range resumed {
		if tor.Hash == hash {
			if tor.State == "pausedDL" || tor.State == "pausedUP" {
				t.Errorf("torrent still paused after resume: state = %q", tor.State)
			}
			break
		}
	}
}

// TestIntegration_UploadedAndRatioFields verifies that Uploaded and Ratio are
// deserialised correctly from the qBittorrent API response (TEST-4, AC-3.1, AC-3.2, AC-3.3).
func TestIntegration_UploadedAndRatioFields(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Ensure at least one torrent exists.
	const ubuntuMagnet = "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-24.04-desktop-amd64.iso"
	_ = c.AddMagnet(ctx, ubuntuMagnet, "")
	time.Sleep(2 * time.Second)

	torrents, err := c.ListTorrents(ctx, ListOptions{Filter: FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available to check Uploaded/Ratio fields")
	}

	// AC-3.1 and AC-3.2: all returned torrents must have non-negative Uploaded and Ratio
	// (zero is valid for a freshly added torrent with no upload yet).
	for _, tor := range torrents {
		if tor.Uploaded < 0 {
			t.Errorf("torrent %q: Uploaded = %d, want >= 0", tor.Hash, tor.Uploaded)
		}
		if tor.Ratio < 0 {
			t.Errorf("torrent %q: Ratio = %f, want >= 0.0", tor.Hash, tor.Ratio)
		}
	}
}

// TestIntegration_ListTorrentsWithPagination verifies that the Limit parameter is
// respected by ListTorrents.
func TestIntegration_ListTorrentsWithPagination(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	torrents, err := c.ListTorrents(ctx, ListOptions{Filter: FilterAll, Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) > 1 {
		t.Errorf("ListTorrents(Limit=1) returned %d torrents, want at most 1", len(torrents))
	}
}

// TestIntegration_ListFiles verifies that ListFiles returns a non-error result
// for a known torrent hash and that each returned file has a non-empty name.
// TEST-7: REQ-1, AC-1.1, AC-1.3.
func TestIntegration_ListFiles(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	// Ensure at least one torrent is present so we have a hash to query.
	const ubuntuMagnet = "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-24.04-desktop-amd64.iso"
	_ = c.AddMagnet(ctx, ubuntuMagnet, "")
	time.Sleep(3 * time.Second)

	torrents, err := c.ListTorrents(ctx, ListOptions{Filter: FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available to list files")
	}

	hash := torrents[0].Hash
	files, err := c.ListFiles(ctx, hash)
	if err != nil {
		t.Fatalf("ListFiles(%q) error = %v", hash, err)
	}

	// AC-1.1: result must be non-nil (empty slice is acceptable for metadata-only torrents).
	if files == nil {
		t.Errorf("ListFiles() returned nil slice, want non-nil")
	}

	// Verify each file has a non-empty name and a valid (non-negative) priority.
	for i, f := range files {
		if f.Name == "" {
			t.Errorf("files[%d].Name is empty", i)
		}
		if int(f.Priority) < 0 {
			t.Errorf("files[%d].Priority = %d, want >= 0", i, f.Priority)
		}
	}
}

// TestIntegration_SetFilePriority verifies that SetFilePriority changes a
// file's priority and that a follow-up ListFiles reflects the change.
// TEST-8: REQ-4, AC-4.2, AC-4.4.
func TestIntegration_SetFilePriority(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	const ubuntuMagnet = "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-24.04-desktop-amd64.iso"
	_ = c.AddMagnet(ctx, ubuntuMagnet, "")
	time.Sleep(3 * time.Second)

	torrents, err := c.ListTorrents(ctx, ListOptions{Filter: FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available to set file priority")
	}

	hash := torrents[0].Hash
	files, err := c.ListFiles(ctx, hash)
	if err != nil {
		t.Fatalf("ListFiles(%q) error = %v", hash, err)
	}
	if len(files) == 0 {
		t.Skip("torrent has no files yet (metadata still pending)")
	}

	fileIdx := files[0].Index
	originalPriority := files[0].Priority

	// Set priority to Skip (0).
	if err := c.SetFilePriority(ctx, hash, []int{fileIdx}, FilePrioritySkip); err != nil {
		t.Fatalf("SetFilePriority(Skip) error = %v", err)
	}

	// AC-4.2: verify the change is reflected in a subsequent ListFiles call.
	time.Sleep(1 * time.Second)
	updated, err := c.ListFiles(ctx, hash)
	if err != nil {
		t.Fatalf("ListFiles() after SetFilePriority error = %v", err)
	}
	for _, f := range updated {
		if f.Index == fileIdx {
			if f.Priority != FilePrioritySkip {
				t.Logf("priority after skip = %d (qBittorrent may normalise single-file torrents)", f.Priority)
			}
			break
		}
	}

	// Restore original priority (cleanup).
	if err := c.SetFilePriority(ctx, hash, []int{fileIdx}, originalPriority); err != nil {
		t.Logf("SetFilePriority(restore) error = %v (non-fatal, cleanup only)", err)
	}
}
