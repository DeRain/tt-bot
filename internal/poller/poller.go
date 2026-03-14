// Package poller periodically checks qBittorrent for newly completed torrents
// and sends Telegram notifications via a Notifier.
package poller

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/home/tt-bot/internal/qbt"
)

// Notifier sends a completion notification to a Telegram chat.
type Notifier interface {
	NotifyCompletion(ctx context.Context, chatID int64, torrent qbt.Torrent) error
}

// Poller periodically checks qBittorrent for newly completed torrents and
// sends notifications via the Notifier.
type Poller struct {
	qbt         qbt.Client
	notifier    Notifier
	interval    time.Duration
	chatIDs     []int64
	knownHashes map[string]bool
	mu          sync.Mutex
}

// New creates a new Poller that polls qBittorrent at the given interval and
// notifies all chatIDs when a torrent completes.
func New(qbtClient qbt.Client, notifier Notifier, interval time.Duration, chatIDs []int64) *Poller {
	ids := make([]int64, len(chatIDs))
	copy(ids, chatIDs)
	return &Poller{
		qbt:         qbtClient,
		notifier:    notifier,
		interval:    interval,
		chatIDs:     ids,
		knownHashes: make(map[string]bool),
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
// On first run it seeds knownHashes with all currently-completed torrents
// to avoid spurious notifications on startup.
func (p *Poller) Run(ctx context.Context) {
	if err := p.seedKnownHashes(ctx); err != nil {
		log.Printf("poller: seed known hashes: %v", err)
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// seedKnownHashes fetches all torrents and marks all currently-completed ones
// as known so they do not trigger notifications on the first poll cycle.
func (p *Poller) seedKnownHashes(ctx context.Context) error {
	torrents, err := p.qbt.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, t := range torrents {
		if t.Progress >= 1.0 {
			p.knownHashes[t.Hash] = true
		}
	}
	return nil
}

// poll performs a single poll iteration: fetches all torrents, notifies for
// newly completed ones, and prunes hashes for deleted torrents.
func (p *Poller) poll(ctx context.Context) {
	torrents, err := p.qbt.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		log.Printf("poller: list torrents: %v", err)
		return
	}

	currentHashes := make(map[string]bool, len(torrents))
	for _, t := range torrents {
		currentHashes[t.Hash] = true
	}

	for _, t := range torrents {
		if t.Progress < 1.0 {
			continue
		}

		p.mu.Lock()
		alreadyKnown := p.knownHashes[t.Hash]
		if !alreadyKnown {
			p.knownHashes[t.Hash] = true
		}
		p.mu.Unlock()

		if alreadyKnown {
			continue
		}

		for _, chatID := range p.chatIDs {
			if err := p.notifier.NotifyCompletion(ctx, chatID, t); err != nil {
				log.Printf("poller: notify chat %d for torrent %q: %v", chatID, t.Name, err)
			}
		}
	}

	p.pruneDeleted(currentHashes)
}

// pruneDeleted removes from knownHashes any hash that is no longer present in
// the current torrent list (i.e. the torrent was deleted from qBittorrent).
func (p *Poller) pruneDeleted(currentHashes map[string]bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for hash := range p.knownHashes {
		if !currentHashes[hash] {
			delete(p.knownHashes, hash)
		}
	}
}
