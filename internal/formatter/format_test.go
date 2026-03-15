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
	if !strings.Contains(msg, "⬇️ Downloading") {
		t.Errorf("expected mapped torrent state in message")
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
		"<< Prev":  "pg:all:2",
		"Page 3/5": "noop",
		"Next >>":  "pg:all:4",
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
		{0, 5, 1},  // zero items → 1 page
		{5, 5, 1},  // exact division
		{6, 5, 2},  // one remainder
		{10, 5, 2}, // exact division
		{11, 5, 3}, // remainder
		{1, 5, 1},  // fewer than one page
		{100, 10, 10},
	}
	for _, c := range cases {
		got := formatter.TotalPages(c.total, c.perPage)
		if got != c.want {
			t.Errorf("TotalPages(%d, %d) = %d, want %d", c.total, c.perPage, got, c.want)
		}
	}
}

// ---- FormatSize ------------------------------------------------------------

func TestFormatSize(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{512 * 1024, "512.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{int64(1.5 * 1024 * 1024 * 1024), "1.5 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}
	for _, c := range cases {
		got := formatter.FormatSize(c.input)
		if got != c.want {
			t.Errorf("FormatSize(%d) = %q, want %q", c.input, got, c.want)
		}
	}
}

// ---- FormatTorrentDetail ---------------------------------------------------

func TestFormatTorrentDetail(t *testing.T) {
	torrent := qbt.Torrent{
		Hash:     "abc123",
		Name:     "Ubuntu 24.04 Desktop AMD64 ISO",
		State:    "downloading",
		Progress: 0.65,
		Size:     2 * 1024 * 1024 * 1024,
		DLSpeed:  5 * 1024 * 1024,
		UPSpeed:  512 * 1024,
		Category: "linux",
	}

	text := formatter.FormatTorrentDetail(torrent)

	if !strings.Contains(text, "Ubuntu 24.04") {
		t.Error("expected full torrent name in detail")
	}
	if !strings.Contains(text, "2.0 GB") {
		t.Error("expected formatted size")
	}
	if !strings.Contains(text, "⬇️ Downloading") {
		t.Error("expected mapped state label")
	}
	if !strings.Contains(text, "linux") {
		t.Error("expected category")
	}
	if !strings.Contains(text, "█") {
		t.Error("expected progress bar")
	}
	if len(text) > formatter.MaxMessageLength {
		t.Errorf("detail text %d chars exceeds limit", len(text))
	}
}

// TEST-2: Uploaded > 0 and Ratio > 0 — REQ-1 (AC-1.1, AC-1.3), REQ-2 (AC-2.1, AC-2.3)
func TestFormatTorrentDetail_UploadedAndRatio_NonZero(t *testing.T) {
	torrent := qbt.Torrent{
		Hash:     "abc123",
		Name:     "Ubuntu 24.04",
		State:    "uploading",
		Progress: 1.0,
		Size:     4 * 1024 * 1024 * 1024,
		DLSpeed:  0,
		UPSpeed:  1024 * 1024,
		Uploaded: 3_435_973_837, // ≈ 3.2 GB
		Ratio:    2.13,
		Category: "linux",
	}

	text := formatter.FormatTorrentDetail(torrent)

	if !strings.Contains(text, "Uploaded: 3.2 GB") {
		t.Errorf("expected 'Uploaded: 3.2 GB' in detail, got:\n%s", text)
	}
	if !strings.Contains(text, "Ratio: 2.13") {
		t.Errorf("expected 'Ratio: 2.13' in detail, got:\n%s", text)
	}

	// AC-1.3: Uploaded line appears between Upload speed and State.
	uploadIdx := strings.Index(text, "Upload:")
	uploadedIdx := strings.Index(text, "Uploaded:")
	stateIdx := strings.Index(text, "State:")
	if uploadedIdx <= uploadIdx {
		t.Errorf("Uploaded line should appear after Upload speed line")
	}
	if stateIdx <= uploadedIdx {
		t.Errorf("State line should appear after Uploaded line")
	}

	// AC-2.3: Ratio line appears immediately after Uploaded line.
	ratioIdx := strings.Index(text, "Ratio:")
	if ratioIdx <= uploadedIdx {
		t.Errorf("Ratio line should appear after Uploaded line")
	}
	if stateIdx <= ratioIdx {
		t.Errorf("State line should appear after Ratio line")
	}
}

// TEST-3: Uploaded == 0 and Ratio == 0.0 — REQ-1 (AC-1.2), REQ-2 (AC-2.2)
func TestFormatTorrentDetail_UploadedAndRatio_Zero(t *testing.T) {
	torrent := qbt.Torrent{
		Hash:     "def456",
		Name:     "Fresh Torrent",
		State:    "downloading",
		Progress: 0.1,
		Size:     1024 * 1024 * 1024,
		DLSpeed:  5 * 1024 * 1024,
		UPSpeed:  0,
		Uploaded: 0,
		Ratio:    0.0,
		Category: "test",
	}

	text := formatter.FormatTorrentDetail(torrent)

	if !strings.Contains(text, "Uploaded: 0 B") {
		t.Errorf("expected 'Uploaded: 0 B' in detail, got:\n%s", text)
	}
	if !strings.Contains(text, "Ratio: 0.00") {
		t.Errorf("expected 'Ratio: 0.00' in detail, got:\n%s", text)
	}
}

func TestFormatTorrentDetail_NoCategory(t *testing.T) {
	torrent := qbt.Torrent{Name: "Test", Category: ""}
	text := formatter.FormatTorrentDetail(torrent)
	if !strings.Contains(text, "none") {
		t.Error("expected 'none' for empty category")
	}
}

func TestFormatTorrentDetail_LongName(t *testing.T) {
	torrent := qbt.Torrent{Name: strings.Repeat("A", 300)}
	text := formatter.FormatTorrentDetail(torrent)
	if len(text) > formatter.MaxMessageLength {
		t.Errorf("detail text %d chars exceeds limit", len(text))
	}
}

// ---- TorrentDetailKeyboard -------------------------------------------------

func TestTorrentDetailKeyboard_AlwaysBothButtons(t *testing.T) {
	states := []string{
		"downloading", "uploading", "seeding",
		"pausedDL", "pausedUP",
		"stalledDL", "stalledUP",
		"stoppedDL", "stoppedUP",
		"queuedDL", "queuedUP",
		"error", "missingFiles",
	}

	hash := strings.Repeat("a", 40)
	for _, state := range states {
		kb := formatter.TorrentDetailKeyboard(hash, "a", 1, state)

		if len(kb) != 2 {
			t.Fatalf("state %q: expected 2 rows, got %d", state, len(kb))
		}

		// Row 1: both Pause and Start buttons side by side.
		row := kb[0]
		if len(row) != 2 {
			t.Fatalf("state %q: expected 2 buttons in row 1, got %d", state, len(row))
		}

		if !strings.Contains(row[0].Text, "Pause") {
			t.Errorf("state %q: expected Pause button first, got %q", state, row[0].Text)
		}
		if !strings.HasPrefix(row[0].CallbackData, "pa:") {
			t.Errorf("state %q: expected pa: prefix, got %q", state, row[0].CallbackData)
		}

		if !strings.Contains(row[1].Text, "Start") {
			t.Errorf("state %q: expected Start button second, got %q", state, row[1].Text)
		}
		if !strings.HasPrefix(row[1].CallbackData, "re:") {
			t.Errorf("state %q: expected re: prefix, got %q", state, row[1].CallbackData)
		}

		// Row 2: Back button.
		if !strings.Contains(kb[1][0].Text, "Back") {
			t.Errorf("state %q: expected Back button, got %q", state, kb[1][0].Text)
		}
	}
}

func TestTorrentDetailKeyboard_CallbackDataUnderLimit(t *testing.T) {
	hash := strings.Repeat("f", 40)
	kb := formatter.TorrentDetailKeyboard(hash, "c", 99, "pausedUP")

	for _, row := range kb {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q (%d bytes) exceeds %d limit",
					btn.CallbackData, len(btn.CallbackData), formatter.MaxCallbackData)
			}
		}
	}
}

// ---- TorrentSelectionKeyboard ----------------------------------------------

func TestTorrentSelectionKeyboard(t *testing.T) {
	torrents := []qbt.Torrent{
		{Hash: strings.Repeat("a", 40), Name: "Torrent A"},
		{Hash: strings.Repeat("b", 40), Name: "Torrent B"},
		{Hash: strings.Repeat("c", 40), Name: "Torrent C"},
	}

	kb := formatter.TorrentSelectionKeyboard(torrents, "a", 1)

	if len(kb) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(kb))
	}

	if !strings.HasPrefix(kb[0][0].Text, "1.") {
		t.Errorf("expected '1.' prefix, got %q", kb[0][0].Text)
	}
	if !strings.HasPrefix(kb[0][0].CallbackData, "sel:a:1:") {
		t.Errorf("unexpected callback: %q", kb[0][0].CallbackData)
	}
	if !strings.HasPrefix(kb[2][0].Text, "3.") {
		t.Errorf("expected '3.' prefix, got %q", kb[2][0].Text)
	}
}

func TestTorrentSelectionKeyboard_Empty(t *testing.T) {
	kb := formatter.TorrentSelectionKeyboard(nil, "a", 1)
	if kb != nil {
		t.Errorf("expected nil keyboard for empty list, got %v", kb)
	}
}

func TestTorrentSelectionKeyboard_CallbackDataUnderLimit(t *testing.T) {
	torrents := []qbt.Torrent{
		{Hash: strings.Repeat("f", 40), Name: "Long Name Torrent"},
	}
	kb := formatter.TorrentSelectionKeyboard(torrents, "c", 99)

	for _, row := range kb {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q (%d bytes) exceeds %d limit",
					btn.CallbackData, len(btn.CallbackData), formatter.MaxCallbackData)
			}
		}
	}
}

// ---- FormatState (status-emojis: TASK-1, TEST-1) ---------------------------

func TestFormatState(t *testing.T) {
	cases := []struct {
		state string
		want  string
	}{
		// All 19 documented states
		{"error", "❌ Error"},
		{"missingFiles", "⚠️ Missing Files"},
		{"uploading", "🌱 Seeding"},
		{"pausedUP", "⏸️ Paused (Seeding)"},
		{"queuedUP", "🕐 Queued (Seeding)"},
		{"stalledUP", "🌱 Seeding (stalled)"},
		{"checkingUP", "🔍 Checking"},
		{"forcedUP", "⏫ Force Seeding"},
		{"allocating", "💾 Allocating"},
		{"downloading", "⬇️ Downloading"},
		{"metaDL", "🔎 Fetching Metadata"},
		{"pausedDL", "⏸️ Paused (Downloading)"},
		{"queuedDL", "🕐 Queued (Downloading)"},
		{"stalledDL", "⬇️ Downloading (stalled)"},
		{"checkingDL", "🔍 Checking"},
		{"forcedDL", "⏬ Force Downloading"},
		{"checkingResumeData", "🔍 Checking"},
		{"moving", "📦 Moving"},
		{"unknown", "❓ Unknown"},
	}
	for _, c := range cases {
		got := formatter.FormatState(c.state)
		if got != c.want {
			t.Errorf("FormatState(%q) = %q, want %q", c.state, got, c.want)
		}
	}
}

func TestFormatState_Fallback(t *testing.T) {
	// Unrecognized state should return "❓ <state>"
	got := formatter.FormatState("newStateFromFuture")
	if got != "❓ newStateFromFuture" {
		t.Errorf("FormatState(unrecognized) = %q, want %q", got, "❓ newStateFromFuture")
	}

	// Empty string fallback
	got = formatter.FormatState("")
	if got != "❓ " {
		t.Errorf("FormatState(\"\") = %q, want %q", got, "❓ ")
	}
}

// ---- FormatTorrentList with mapped states (status-emojis: TASK-2, TEST-2) --

func TestFormatTorrentList_UsesMappedState(t *testing.T) {
	torrents := []qbt.Torrent{
		makeTorrent("Test Torrent", 0.5, 1024, 512, "stalledDL"),
	}
	msg := formatter.FormatTorrentList(torrents, 1, 1)

	if !strings.Contains(msg, "⬇️ Downloading (stalled)") {
		t.Errorf("expected mapped state label in list, got: %s", msg)
	}
	// Raw state should not appear (except as substring of the label — "stalledDL" won't match "Downloading (stalled)")
	if strings.Contains(msg, "stalledDL") {
		t.Errorf("raw state 'stalledDL' should not appear in list output")
	}
}

func TestFormatTorrentDetail_UsesMappedState(t *testing.T) {
	torrent := qbt.Torrent{
		Hash:     "abc123",
		Name:     "Test Torrent",
		State:    "pausedUP",
		Progress: 1.0,
		Size:     1024 * 1024 * 1024,
		Category: "test",
	}
	text := formatter.FormatTorrentDetail(torrent)

	if !strings.Contains(text, "⏸️ Paused (Seeding)") {
		t.Errorf("expected mapped state label in detail, got: %s", text)
	}
	if strings.Contains(text, "pausedUP") {
		t.Errorf("raw state 'pausedUP' should not appear in detail output")
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
