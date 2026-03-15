package qbt

import (
	"context"
	"io"
)

// Client defines the operations available against the qBittorrent Web API v2.
// All methods accept a context so callers can apply deadlines and cancellation.
type Client interface {
	// Login authenticates with qBittorrent and stores the session cookie.
	// It must be called before any other method, though implementations may
	// call it automatically on receiving a 403 response.
	Login(ctx context.Context) error

	// AddMagnet adds a torrent by magnet URI and assigns it to the given category.
	// An empty category string leaves the torrent uncategorised.
	AddMagnet(ctx context.Context, magnet string, category string) error

	// AddTorrentFile uploads a .torrent file and assigns it to the given category.
	// filename is used as the MIME part filename; data provides the file bytes.
	AddTorrentFile(ctx context.Context, filename string, data io.Reader, category string) error

	// ListTorrents returns the list of torrents matching opts.
	ListTorrents(ctx context.Context, opts ListOptions) ([]Torrent, error)

	// Categories returns all categories configured in qBittorrent, sorted by name.
	Categories(ctx context.Context) ([]Category, error)

	// PauseTorrents pauses one or more torrents identified by info-hash.
	PauseTorrents(ctx context.Context, hashes []string) error

	// ResumeTorrents resumes one or more torrents identified by info-hash.
	ResumeTorrents(ctx context.Context, hashes []string) error

	// DeleteTorrents removes one or more torrents identified by info-hash.
	// If deleteFiles is true, the associated downloaded data is also deleted from disk.
	DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error

	// ListFiles returns the files contained within the torrent identified by hash.
	ListFiles(ctx context.Context, hash string) ([]TorrentFile, error)

	// SetFilePriority sets the download priority for the given file indices within
	// the torrent identified by hash. fileIndices must be non-negative integers
	// matching the Index field of TorrentFile entries returned by ListFiles.
	SetFilePriority(ctx context.Context, hash string, fileIndices []int, priority FilePriority) error
}
