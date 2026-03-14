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
