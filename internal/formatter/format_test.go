package formatter_test

import (
	"strings"
	"testing"

	"github.com/home/tt-bot/internal/formatter"
	"github.com/home/tt-bot/internal/qbt"
)

// ---- helpers ---------------------------------------------------------------

func makeTorrent(name string, progress float64, dlSpeed, upSpeed int64, state string) qbt.Torrent {
	return qbt.Torrent{
		Hash:     "abc123",
		Name:     name,
		Progress: progress,
		DLSpeed:  dlSpeed,
		UPSpeed:  upSpeed,
		State:    state,
	}
}

func fiveTorrents() []qbt.Torrent {
	return []qbt.Torrent{
		makeTorrent("Ubuntu 24.04 Desktop AMD64 ISO", 0.6, 2*1024*1024, 512*1024, "downloading"),
		makeTorrent("Fedora Workstation 40", 0.9, 500*1024, 100*1024, "downloading"),
		makeTorrent("Debian 12 netinst", 1.0, 0, 1024*1024, "seeding"),
		makeTorrent("Arch Linux 2024.01.01", 0.1, 10*1024*1024, 0, "downloading"),
		makeTorrent("openSUSE Tumbleweed DVD", 0.45, 750*1024, 200*1024, "downloading"),
	}
}

// ---- FormatTorrentList -----------------------------------------------------

func TestFormatTorrentList_FiveTorrents_UnderLimit(t *testing.T) {
	torrents := fiveTorrents()
	msg := formatter.FormatTorrentList(torrents, 1, 3)

	if len(msg) >= formatter.MaxMessageLength {
		t.Errorf("message length %d >= MaxMessageLength %d", len(msg), formatter.MaxMessageLength)
	}
	if !strings.Contains(msg, "page 1/3") {
		t.Errorf("expected page header in message, got: %q", msg)
	}
}

func TestFormatTorrentList_Empty_ReturnsNotFound(t *testing.T) {
	msg := formatter.FormatTorrentList(nil, 1, 1)
	if msg != "No torrents found." {
		t.Errorf("expected 'No torrents found.', got %q", msg)
	}

	msg2 := formatter.FormatTorrentList([]qbt.Torrent{}, 1, 1)
	if msg2 != "No torrents found." {
		t.Errorf("expected 'No torrents found.' for empty slice, got %q", msg2)
	}
}

func TestFormatTorrentList_WorstCaseLongNames_UnderLimit(t *testing.T) {
	// Build 5 torrents whose names are exactly 40 runes — the truncation boundary.
	longName := strings.Repeat("A", 40)
	torrents := make([]qbt.Torrent, formatter.TorrentsPerPage)
	for i := range torrents {
		torrents[i] = makeTorrent(longName, 0.5, 999*1024*1024, 999*1024*1024, "downloading")
	}

	msg := formatter.FormatTorrentList(torrents, 1, 1)
	if len(msg) >= formatter.MaxMessageLength {
		t.Errorf("worst-case message length %d >= MaxMessageLength %d", len(msg), formatter.MaxMessageLength)
	}
}

func TestFormatTorrentList_ContainsTorrentDetails(t *testing.T) {
	torrents := []qbt.Torrent{
		makeTorrent("Ubuntu 24.04", 0.6, 2*1024*1024, 512*1024, "downloading"),
	}
	msg := formatter.FormatTorrentList(torrents, 1, 1)

	if !strings.Contains(msg, "Ubuntu 24.04") {
		t.Errorf("expected torrent name in message")
	}
	if !strings.Contains(msg, "downloading") {
		t.Errorf("expected torrent state in message")
	}
	// Progress bar should contain block characters.
	if !strings.Contains(msg, "█") {
		t.Errorf("expected progress bar in message")
	}
}

// ---- FormatSpeed -----------------------------------------------------------

func TestFormatSpeed_BytesPerSec(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{0, "0 B/s"},
		{512, "512 B/s"},
		{1023, "1023 B/s"},
	}
	for _, c := range cases {
		got := formatter.FormatSpeed(c.input)
		if got != c.want {
			t.Errorf("FormatSpeed(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFormatSpeed_KilobytesPerSec(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{1024, "1.0 KB/s"},
		{512 * 1024, "512.0 KB/s"},
		{1023 * 1024, "1023.0 KB/s"},
	}
	for _, c := range cases {
		got := formatter.FormatSpeed(c.input)
		if got != c.want {
			t.Errorf("FormatSpeed(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFormatSpeed_MegabytesPerSec(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{1024 * 1024, "1.0 MB/s"},
		{2*1024*1024 + 100*1024, "2.1 MB/s"},
		{10 * 1024 * 1024, "10.0 MB/s"},
	}
	for _, c := range cases {
		got := formatter.FormatSpeed(c.input)
		if got != c.want {
			t.Errorf("FormatSpeed(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ---- FormatProgress --------------------------------------------------------

func TestFormatProgress(t *testing.T) {
	cases := []struct {
		input float64
		want  string
	}{
		{0.0, "░░░░░░░░░░ 0%"},
		{0.5, "█████░░░░░ 50%"},
		{1.0, "██████████ 100%"},
		{0.1, "█░░░░░░░░░ 10%"},
		{0.9, "█████████░ 90%"},
	}
	for _, c := range cases {
		got := formatter.FormatProgress(c.input)
		if got != c.want {
			t.Errorf("FormatProgress(%.1f) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFormatProgress_EdgeValues(t *testing.T) {
	// Values outside [0,1] should be clamped.
	neg := formatter.FormatProgress(-0.5)
	if !strings.HasPrefix(neg, "░░░░░░░░░░") {
		t.Errorf("FormatProgress(-0.5) should clamp to 0, got %q", neg)
	}

	over := formatter.FormatProgress(1.5)
	if !strings.HasPrefix(over, "██████████") {
		t.Errorf("FormatProgress(1.5) should clamp to 100, got %q", over)
	}
}

// ---- PaginationKeyboard ----------------------------------------------------

func TestPaginationKeyboard_FirstPage_NoPrev(t *testing.T) {
	kb := formatter.PaginationKeyboard(1, 5, "all")
	if len(kb) != 1 {
		t.Fatalf("expected 1 row, got %d", len(kb))
	}
	row := kb[0]
	for _, btn := range row {
		if btn.Text == "<< Prev" {
			t.Error("first page should not have a Prev button")
		}
	}
	// Should have "Next >>"
	hasNext := false
	for _, btn := range row {
		if btn.Text == "Next >>" {
			hasNext = true
		}
	}
	if !hasNext {
		t.Error("first page should have a Next button when totalPages > 1")
	}
}

func TestPaginationKeyboard_LastPage_NoNext(t *testing.T) {
	kb := formatter.PaginationKeyboard(5, 5, "act")
	if len(kb) != 1 {
		t.Fatalf("expected 1 row, got %d", len(kb))
	}
	row := kb[0]
	for _, btn := range row {
		if btn.Text == "Next >>" {
			t.Error("last page should not have a Next button")
		}
	}
	// Should have "<< Prev"
	hasPrev := false
	for _, btn := range row {
		if btn.Text == "<< Prev" {
			hasPrev = true
		}
	}
	if !hasPrev {
		t.Error("last page should have a Prev button")
	}
}

func TestPaginationKeyboard_MiddlePage_BothButtons(t *testing.T) {
	kb := formatter.PaginationKeyboard(3, 5, "all")
	if len(kb) != 1 {
		t.Fatalf("expected 1 row, got %d", len(kb))
	}
	row := kb[0]
	if len(row) != 3 {
		t.Fatalf("middle page should have 3 buttons, got %d", len(row))
	}

	wantCallbacks := map[string]string{
		"<< Prev":   "pg:all:2",
		"Page 3/5":  "noop",
		"Next >>":   "pg:all:4",
	}
	for _, btn := range row {
		want, ok := wantCallbacks[btn.Text]
		if !ok {
			t.Errorf("unexpected button %q", btn.Text)
			continue
		}
		if btn.CallbackData != want {
			t.Errorf("button %q: callback = %q, want %q", btn.Text, btn.CallbackData, want)
		}
	}
}

func TestPaginationKeyboard_CallbackDataUnderLimit(t *testing.T) {
	kb := formatter.PaginationKeyboard(999, 9999, "all")
	for _, row := range kb {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q exceeds %d bytes", btn.CallbackData, formatter.MaxCallbackData)
			}
		}
	}
}

// ---- CategoryKeyboard ------------------------------------------------------

func TestCategoryKeyboard_Normal(t *testing.T) {
	cats := []qbt.Category{
		{Name: "movies", SavePath: "/dl/movies"},
		{Name: "tv", SavePath: "/dl/tv"},
	}
	kb := formatter.CategoryKeyboard(cats)
	if len(kb) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(kb))
	}
	if kb[0][0].CallbackData != "cat:movies" {
		t.Errorf("unexpected callback: %q", kb[0][0].CallbackData)
	}
	if kb[1][0].CallbackData != "cat:tv" {
		t.Errorf("unexpected callback: %q", kb[1][0].CallbackData)
	}
}

func TestCategoryKeyboard_Empty(t *testing.T) {
	kb := formatter.CategoryKeyboard(nil)
	if len(kb) != 1 {
		t.Fatalf("expected 1 row for empty list, got %d", len(kb))
	}
	if kb[0][0].Text != "No category" {
		t.Errorf("expected 'No category' button, got %q", kb[0][0].Text)
	}
	if kb[0][0].CallbackData != "cat:" {
		t.Errorf("expected 'cat:' callback, got %q", kb[0][0].CallbackData)
	}
}

func TestCategoryKeyboard_LongNameTruncated(t *testing.T) {
	// A category name that would push "cat:" + name past 64 bytes.
	longName := strings.Repeat("x", 70)
	cats := []qbt.Category{{Name: longName}}
	kb := formatter.CategoryKeyboard(cats)

	btn := kb[0][0]
	if len(btn.CallbackData) > formatter.MaxCallbackData {
		t.Errorf("callback data %d bytes exceeds %d limit", len(btn.CallbackData), formatter.MaxCallbackData)
	}
}

func TestCategoryKeyboard_CallbackDataUnderLimit(t *testing.T) {
	cats := []qbt.Category{
		{Name: strings.Repeat("a", 100)},
		{Name: "short"},
	}
	kb := formatter.CategoryKeyboard(cats)
	for _, row := range kb {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q exceeds %d bytes", btn.CallbackData, formatter.MaxCallbackData)
			}
		}
	}
}

// ---- TotalPages ------------------------------------------------------------

func TestTotalPages(t *testing.T) {
	cases := []struct {
		total, perPage, want int
	}{
		{0, 5, 1},   // zero items → 1 page
		{5, 5, 1},   // exact division
		{6, 5, 2},   // one remainder
		{10, 5, 2},  // exact division
		{11, 5, 3},  // remainder
		{1, 5, 1},   // fewer than one page
		{100, 10, 10},
	}
	for _, c := range cases {
		got := formatter.TotalPages(c.total, c.perPage)
		if got != c.want {
			t.Errorf("TotalPages(%d, %d) = %d, want %d", c.total, c.perPage, got, c.want)
		}
	}
}

// ---- All callback data must never exceed MaxCallbackData -------------------

func TestAllCallbackDataUnderLimit(t *testing.T) {
	// Category keyboard with max-length name.
	cats := []qbt.Category{{Name: strings.Repeat("z", 100)}}
	for _, row := range formatter.CategoryKeyboard(cats) {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("CategoryKeyboard callback %q exceeds limit", btn.CallbackData)
			}
		}
	}

	// Pagination keyboard with large page numbers.
	for _, row := range formatter.PaginationKeyboard(9999, 99999, "all") {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("PaginationKeyboard callback %q exceeds limit", btn.CallbackData)
			}
		}
	}
}
