package poller

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/home/tt-bot/internal/qbt"
)

// mockQBT is a thread-safe fake qbt.Client whose torrent list can be swapped
// between test steps.
type mockQBT struct {
	calls    int
	torrents []qbt.Torrent
	mu       sync.Mutex
}

func (m *mockQBT) setTorrents(ts []qbt.Torrent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.torrents = ts
}

func (m *mockQBT) Login(_ context.Context) error { return nil }

func (m *mockQBT) AddMagnet(_ context.Context, _ string, _ string) error { return nil }

func (m *mockQBT) AddTorrentFile(_ context.Context, _ string, _ io.Reader, _ string) error {
	return nil
}

func (m *mockQBT) ListTorrents(_ context.Context, _ qbt.ListOptions) ([]qbt.Torrent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	result := make([]qbt.Torrent, len(m.torrents))
	copy(result, m.torrents)
	return result, nil
}

func (m *mockQBT) Categories(_ context.Context) ([]qbt.Category, error) {
	return nil, nil
}

func (m *mockQBT) PauseTorrents(_ context.Context, _ []string) error  { return nil }
func (m *mockQBT) ResumeTorrents(_ context.Context, _ []string) error { return nil }

// notification captures a single call to NotifyCompletion.
type notification struct {
	chatID  int64
	torrent qbt.Torrent
}

// mockNotifier records all notifications sent.
type mockNotifier struct {
	notifications []notification
	mu            sync.Mutex
}

func (n *mockNotifier) NotifyCompletion(_ context.Context, chatID int64, torrent qbt.Torrent) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.notifications = append(n.notifications, notification{chatID: chatID, torrent: torrent})
	return nil
}

func (n *mockNotifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.notifications)
}

func (n *mockNotifier) all() []notification {
	n.mu.Lock()
	defer n.mu.Unlock()
	result := make([]notification, len(n.notifications))
	copy(result, n.notifications)
	return result
}

// torrent helpers

func completedTorrent(hash, name string) qbt.Torrent {
	return qbt.Torrent{Hash: hash, Name: name, Progress: 1.0}
}

func incompleteTorrent(hash, name string) qbt.Torrent {
	return qbt.Torrent{Hash: hash, Name: name, Progress: 0.5}
}

// waitFor spins until cond() returns true or the deadline passes.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return false
}

// TestPoller_SeedsKnownHashes_NoNotificationOnStart verifies that torrents
// which are already complete when the poller starts do not trigger
// notifications.
func TestPoller_SeedsKnownHashes_NoNotificationOnStart(t *testing.T) {
	mock := &mockQBT{
		torrents: []qbt.Torrent{
			completedTorrent("abc", "TorrentA"),
			completedTorrent("def", "TorrentB"),
		},
	}
	notifier := &mockNotifier{}
	chatIDs := []int64{111}

	p := New(mock, notifier, 10*time.Millisecond, chatIDs)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go p.Run(ctx)
	<-ctx.Done()

	if notifier.count() != 0 {
		t.Errorf("expected 0 notifications on startup, got %d", notifier.count())
	}
}

// TestPoller_NotifiesOnNewCompletion verifies that a torrent which transitions
// from incomplete to complete triggers exactly one notification per chatID.
func TestPoller_NotifiesOnNewCompletion(t *testing.T) {
	mock := &mockQBT{
		torrents: []qbt.Torrent{
			incompleteTorrent("xyz", "TorrentX"),
		},
	}
	notifier := &mockNotifier{}
	chatIDs := []int64{111}

	p := New(mock, notifier, 10*time.Millisecond, chatIDs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Run(ctx)

	// Wait for the seed pass to complete (at least one ListTorrents call).
	if !waitFor(t, 500*time.Millisecond, func() bool {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		return mock.calls >= 1
	}) {
		t.Fatal("seed pass did not complete in time")
	}

	// Mark the torrent as complete and wait for notification.
	mock.setTorrents([]qbt.Torrent{completedTorrent("xyz", "TorrentX")})

	if !waitFor(t, 500*time.Millisecond, func() bool {
		return notifier.count() >= 1
	}) {
		t.Fatal("expected notification after torrent completed, got none")
	}

	got := notifier.all()
	if len(got) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(got))
	}
	if got[0].chatID != 111 {
		t.Errorf("expected chatID 111, got %d", got[0].chatID)
	}
	if got[0].torrent.Hash != "xyz" {
		t.Errorf("expected hash xyz, got %s", got[0].torrent.Hash)
	}
}

// TestPoller_NoDuplicateNotifications verifies that a torrent which stays
// complete across multiple polls triggers only a single notification.
func TestPoller_NoDuplicateNotifications(t *testing.T) {
	mock := &mockQBT{
		torrents: []qbt.Torrent{
			incompleteTorrent("dup", "DupTorrent"),
		},
	}
	notifier := &mockNotifier{}
	chatIDs := []int64{111}

	p := New(mock, notifier, 10*time.Millisecond, chatIDs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Run(ctx)

	// Wait for seed pass.
	if !waitFor(t, 500*time.Millisecond, func() bool {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		return mock.calls >= 1
	}) {
		t.Fatal("seed pass did not complete in time")
	}

	// Mark complete and let several poll cycles run.
	mock.setTorrents([]qbt.Torrent{completedTorrent("dup", "DupTorrent")})

	if !waitFor(t, 500*time.Millisecond, func() bool {
		return notifier.count() >= 1
	}) {
		t.Fatal("expected at least 1 notification")
	}

	// Let a few more ticks fire to ensure no duplicates accumulate.
	time.Sleep(60 * time.Millisecond)

	if notifier.count() != 1 {
		t.Errorf("expected exactly 1 notification, got %d", notifier.count())
	}
}

// TestPoller_PrunesDeletedTorrents verifies that when a torrent disappears
// from the list its hash is pruned, so if the same torrent reappears as
// complete later it triggers a fresh notification.
func TestPoller_PrunesDeletedTorrents(t *testing.T) {
	// Start with one complete torrent.
	mock := &mockQBT{
		torrents: []qbt.Torrent{
			completedTorrent("gone", "GoneTorrent"),
		},
	}
	notifier := &mockNotifier{}
	chatIDs := []int64{111}

	p := New(mock, notifier, 10*time.Millisecond, chatIDs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Run(ctx)

	// Wait for seed pass so "gone" is in knownHashes.
	if !waitFor(t, 500*time.Millisecond, func() bool {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		return mock.calls >= 1
	}) {
		t.Fatal("seed pass did not complete in time")
	}

	// Let a poll cycle run with the torrent still present (still in knownHashes, no notification).
	time.Sleep(30 * time.Millisecond)
	if notifier.count() != 0 {
		t.Fatalf("expected 0 notifications before deletion, got %d", notifier.count())
	}

	// Remove the torrent — prune should remove its hash.
	mock.setTorrents([]qbt.Torrent{})

	// Wait for a poll cycle to run the prune.
	time.Sleep(30 * time.Millisecond)

	// Verify hash was pruned by checking the internal map.
	p.mu.Lock()
	_, stillKnown := p.knownHashes["gone"]
	p.mu.Unlock()
	if stillKnown {
		t.Error("expected hash 'gone' to be pruned after torrent deletion")
	}

	// Now re-add the torrent as complete; should trigger a notification.
	mock.setTorrents([]qbt.Torrent{completedTorrent("gone", "GoneTorrent")})

	if !waitFor(t, 500*time.Millisecond, func() bool {
		return notifier.count() >= 1
	}) {
		t.Fatal("expected notification after re-appearing completed torrent, got none")
	}
}

// TestPoller_NotifiesMultipleUsers verifies that all chatIDs receive a
// notification when a torrent completes.
func TestPoller_NotifiesMultipleUsers(t *testing.T) {
	mock := &mockQBT{
		torrents: []qbt.Torrent{
			incompleteTorrent("multi", "MultiTorrent"),
		},
	}
	notifier := &mockNotifier{}
	chatIDs := []int64{111, 222, 333}

	p := New(mock, notifier, 10*time.Millisecond, chatIDs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Run(ctx)

	// Wait for seed pass.
	if !waitFor(t, 500*time.Millisecond, func() bool {
		mock.mu.Lock()
		defer mock.mu.Unlock()
		return mock.calls >= 1
	}) {
		t.Fatal("seed pass did not complete in time")
	}

	mock.setTorrents([]qbt.Torrent{completedTorrent("multi", "MultiTorrent")})

	if !waitFor(t, 500*time.Millisecond, func() bool {
		return notifier.count() >= 3
	}) {
		t.Fatalf("expected 3 notifications (one per user), got %d", notifier.count())
	}

	// Collect seen chatIDs.
	seen := make(map[int64]bool)
	for _, n := range notifier.all() {
		seen[n.chatID] = true
	}

	for _, id := range chatIDs {
		if !seen[id] {
			t.Errorf("chatID %d did not receive a notification", id)
		}
	}
}
