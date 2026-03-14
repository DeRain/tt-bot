// Package qbt provides a thin client for the qBittorrent Web API v2.
package qbt

// TorrentFilter specifies which torrents to retrieve from qBittorrent.
type TorrentFilter string

const (
	// FilterAll returns all torrents regardless of state.
	FilterAll TorrentFilter = "all"
	// FilterActive returns only torrents that are actively downloading or seeding.
	FilterActive TorrentFilter = "active"
)

// Torrent represents a single torrent item as returned by the qBittorrent API.
// All fields are read-only; the struct is intended to be used as an immutable value.
type Torrent struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	State        string  `json:"state"`
	Progress     float64 `json:"progress"`
	Size         int64   `json:"size"`
	DLSpeed      int64   `json:"dlspeed"`
	UPSpeed      int64   `json:"upspeed"`
	ETA          int64   `json:"eta"`
	Category     string  `json:"category"`
	CompletionOn int64   `json:"completion_on"`
	AddedOn      int64   `json:"added_on"`
}

// Category represents a torrent category configured in qBittorrent.
type Category struct {
	Name     string `json:"name"`
	SavePath string `json:"savePath"`
}

// ListOptions controls filtering and pagination when listing torrents.
type ListOptions struct {
	// Filter selects which subset of torrents to return.
	Filter TorrentFilter
	// Limit is the maximum number of torrents to return. Zero means no limit.
	Limit int
	// Offset is the zero-based index of the first torrent to return.
	Offset int
}
