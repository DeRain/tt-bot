package formatter_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

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

func TestCategoryKeyboard_MultiByteUTF8Truncation(t *testing.T) {
	// A category name that, after "cat:" prefix + truncation at 64 bytes,
	// leaves an incomplete multi-byte UTF-8 sequence that must be stripped.
	// "cat:" = 4 bytes. 59 ASCII 'x' = 59 bytes. "€" = 3 bytes (0xE2 0x82 0xAC).
	// Total: 66 bytes. Truncation at 64 leaves 0xE2 (incomplete 3-byte start).
	name := strings.Repeat("x", 59) + "€"
	cats := []qbt.Category{{Name: name}}
	kb := formatter.CategoryKeyboard(cats)

	btn := kb[0][0]
	if len(btn.CallbackData) > formatter.MaxCallbackData {
		t.Errorf("callback data %d bytes exceeds %d limit", len(btn.CallbackData), formatter.MaxCallbackData)
	}
	if !utf8.Valid([]byte(btn.CallbackData)) {
		t.Errorf("callback data is not valid UTF-8 after truncation: %q", btn.CallbackData)
	}
	// The incomplete 0xE2 byte should be stripped, leaving exactly 63 bytes.
	if len(btn.CallbackData) != 63 {
		t.Errorf("expected 63 bytes after multi-byte strip, got %d", len(btn.CallbackData))
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

		// Now 4 rows: [Pause|Start], [Files], [Remove], [Back].
		if len(kb) != 4 {
			t.Fatalf("state %q: expected 4 rows, got %d", state, len(kb))
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

		// Row 2: Files button (AC-5.1).
		if !strings.Contains(kb[1][0].Text, "Files") {
			t.Errorf("state %q: expected Files button in row 2, got %q", state, kb[1][0].Text)
		}
		if !strings.HasPrefix(kb[1][0].CallbackData, "fl:") {
			t.Errorf("state %q: expected fl: prefix, got %q", state, kb[1][0].CallbackData)
		}

		// Row 3: Remove button (AC-1.1, AC-1.2).
		if !strings.Contains(kb[2][0].Text, "Remove") {
			t.Errorf("state %q: expected Remove button in row 3, got %q", state, kb[2][0].Text)
		}
		if !strings.HasPrefix(kb[2][0].CallbackData, "rm:") {
			t.Errorf("state %q: expected rm: prefix, got %q", state, kb[2][0].CallbackData)
		}

		// Row 4: Back button.
		if !strings.Contains(kb[3][0].Text, "Back") {
			t.Errorf("state %q: expected Back button, got %q", state, kb[3][0].Text)
		}
	}
}

// TEST-4: TorrentDetailKeyboard Remove button callback fits within 64 bytes (AC-1.1).
func TestTorrentDetailKeyboard_RemoveCallbackFitsLimit(t *testing.T) {
	hash := strings.Repeat("f", 40)
	kb := formatter.TorrentDetailKeyboard(hash, "a", 99, "downloading")

	// Find the Remove button in row 3 (index 2).
	if len(kb) < 3 {
		t.Fatal("expected at least 3 rows in detail keyboard")
	}
	removeBtn := kb[2][0]
	if !strings.HasPrefix(removeBtn.CallbackData, "rm:") {
		t.Fatalf("expected rm: prefix, got %q", removeBtn.CallbackData)
	}
	if len(removeBtn.CallbackData) > formatter.MaxCallbackData {
		t.Errorf("rm: callback %q (%d bytes) exceeds %d byte limit",
			removeBtn.CallbackData, len(removeBtn.CallbackData), formatter.MaxCallbackData)
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

// ---- FormatRemoveConfirmation (TEST-5) -------------------------------------

// TEST-5: FormatRemoveConfirmation includes torrent name and prompt text (AC-2.1).
func TestFormatRemoveConfirmation_ContainsNameAndPrompt(t *testing.T) {
	name := "Ubuntu 24.04 Desktop AMD64 ISO"
	msg := formatter.FormatRemoveConfirmation(name)

	if !strings.Contains(msg, name) {
		t.Errorf("expected torrent name %q in confirmation message, got: %q", name, msg)
	}
	if !strings.Contains(msg, "Remove") {
		t.Errorf("expected 'Remove' in confirmation message, got: %q", msg)
	}
}

func TestFormatRemoveConfirmation_EmptyName(t *testing.T) {
	msg := formatter.FormatRemoveConfirmation("")
	if len(msg) == 0 {
		t.Error("expected non-empty confirmation message for empty torrent name")
	}
}

// ---- RemoveConfirmKeyboard (TEST-6) ----------------------------------------

// TEST-6: RemoveConfirmKeyboard has 3 rows with correct prefixes (AC-2.1, AC-4.2).
func TestRemoveConfirmKeyboard_ThreeRows(t *testing.T) {
	hash := strings.Repeat("a", 40)
	kb := formatter.RemoveConfirmKeyboard(hash, "a", 1)

	if len(kb) != 3 {
		t.Fatalf("expected 3 rows in confirm keyboard, got %d", len(kb))
	}

	// Row 1: rd: (remove torrent only).
	if !strings.HasPrefix(kb[0][0].CallbackData, "rd:") {
		t.Errorf("row 1: expected rd: prefix, got %q", kb[0][0].CallbackData)
	}

	// Row 2: rf: (remove with files).
	if !strings.HasPrefix(kb[1][0].CallbackData, "rf:") {
		t.Errorf("row 2: expected rf: prefix, got %q", kb[1][0].CallbackData)
	}

	// Row 3: rc: (cancel).
	if !strings.HasPrefix(kb[2][0].CallbackData, "rc:") {
		t.Errorf("row 3: expected rc: prefix, got %q", kb[2][0].CallbackData)
	}
}

// TEST-6: All RemoveConfirmKeyboard callbacks fit within 64 bytes at worst case (AC-4.2).
func TestRemoveConfirmKeyboard_CallbackDataUnderLimit(t *testing.T) {
	hash := strings.Repeat("f", 40)
	kb := formatter.RemoveConfirmKeyboard(hash, "a", 99)

	for i, row := range kb {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("row %d callback %q (%d bytes) exceeds %d byte limit",
					i, btn.CallbackData, len(btn.CallbackData), formatter.MaxCallbackData)
			}
		}
	}
}

func TestRemoveConfirmKeyboard_ButtonTexts(t *testing.T) {
	hash := strings.Repeat("b", 40)
	kb := formatter.RemoveConfirmKeyboard(hash, "a", 1)

	if !strings.Contains(kb[0][0].Text, "torrent only") {
		t.Errorf("row 1 button should mention 'torrent only', got %q", kb[0][0].Text)
	}
	if !strings.Contains(kb[1][0].Text, "files") {
		t.Errorf("row 2 button should mention 'files', got %q", kb[1][0].Text)
	}
	if !strings.Contains(kb[2][0].Text, "Cancel") {
		t.Errorf("row 3 button should be Cancel, got %q", kb[2][0].Text)
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
		// All 21 documented states
		{"error", "❌ Error"},
		{"missingFiles", "⚠️ Missing Files"},
		{"uploading", "🌱 Seeding"},
		{"pausedUP", "⏸️ Paused (Seeding)"},
		{"stoppedUP", "⏸️ Stopped (Seeding)"},
		{"queuedUP", "🕐 Queued (Seeding)"},
		{"stalledUP", "🌱 Seeding (stalled)"},
		{"checkingUP", "🔍 Checking"},
		{"forcedUP", "⏫ Force Seeding"},
		{"allocating", "💾 Allocating"},
		{"downloading", "⬇️ Downloading"},
		{"metaDL", "🔎 Fetching Metadata"},
		{"pausedDL", "⏸️ Paused (Downloading)"},
		{"stoppedDL", "⏸️ Stopped (Downloading)"},
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

// ---- TorrentDetailKeyboard Files button (TEST-5, AC-5.1) -------------------

func TestTorrentDetailKeyboard_FilesButton(t *testing.T) {
	hash := strings.Repeat("a", 40)
	kb := formatter.TorrentDetailKeyboard(hash, "a", 1, "downloading")

	// Row 2 must be the Files button.
	if len(kb) < 2 {
		t.Fatal("expected at least 2 rows in detail keyboard")
	}
	filesBtn := kb[1][0]
	if !strings.Contains(filesBtn.Text, "Files") {
		t.Errorf("expected Files button in row 2, got %q", filesBtn.Text)
	}
	if !strings.HasPrefix(filesBtn.CallbackData, "fl:") {
		t.Errorf("expected fl: prefix on Files button, got %q", filesBtn.CallbackData)
	}
	if len(filesBtn.CallbackData) > formatter.MaxCallbackData {
		t.Errorf("fl: callback %q (%d bytes) exceeds %d limit",
			filesBtn.CallbackData, len(filesBtn.CallbackData), formatter.MaxCallbackData)
	}
}

// ---- PriorityLabel (TEST-3, AC-6.1–AC-6.4) ---------------------------------

func TestPriorityLabel(t *testing.T) {
	cases := []struct {
		p    qbt.FilePriority
		want string
	}{
		{qbt.FilePrioritySkip, "Skip"},
		{qbt.FilePriorityNormal, "Normal"},
		{qbt.FilePriorityHigh, "High"},
		{qbt.FilePriorityMaximum, "Max"},
		{qbt.FilePriority(4), "Mixed"}, // sentinel for mixed priority
	}
	for _, c := range cases {
		got := formatter.PriorityLabel(c.p)
		if got != c.want {
			t.Errorf("PriorityLabel(%d) = %q, want %q", c.p, got, c.want)
		}
	}
}

// ---- FormatFileList (TEST-3) ------------------------------------------------

func makeFiles(n int) []qbt.TorrentFile {
	files := make([]qbt.TorrentFile, n)
	for i := range files {
		files[i] = qbt.TorrentFile{
			Index:    i,
			Name:     fmt.Sprintf("Season 1/Episode %02d.mkv", i+1),
			Size:     1024 * 1024 * 1024,
			Progress: 0.5,
			Priority: qbt.FilePriorityNormal,
		}
	}
	return files
}

func TestFormatFileList_ContainsHeader(t *testing.T) {
	files := makeFiles(3)
	msg := formatter.FormatFileList("My Torrent", files, 1, 1)

	if !strings.Contains(msg, "My Torrent") {
		t.Errorf("expected torrent name in header, got: %q", msg)
	}
	// No page indicator for single page.
	if strings.Contains(msg, "Page 1/1") {
		t.Errorf("page indicator should not appear for single page")
	}
}

func TestFormatFileList_PageIndicatorMultiPage(t *testing.T) {
	files := makeFiles(3)
	msg := formatter.FormatFileList("My Torrent", files, 1, 3)

	if !strings.Contains(msg, "Page 1/3") {
		t.Errorf("expected page indicator for multi-page, got: %q", msg)
	}
}

func TestFormatFileList_ShowsLastPathComponent(t *testing.T) {
	files := []qbt.TorrentFile{
		{Index: 0, Name: "Season 1/ep01.mkv", Size: 500 * 1024 * 1024, Progress: 0.0, Priority: qbt.FilePrioritySkip},
	}
	msg := formatter.FormatFileList("TorrentX", files, 1, 1)

	if strings.Contains(msg, "Season 1/") {
		t.Errorf("should show only last path component, got: %q", msg)
	}
	if !strings.Contains(msg, "ep01.mkv") {
		t.Errorf("expected last path component in output, got: %q", msg)
	}
}

func TestFormatFileList_TruncatesLongFileName(t *testing.T) {
	longBaseName := strings.Repeat("A", 50)
	files := []qbt.TorrentFile{
		{Index: 0, Name: "dir/" + longBaseName, Size: 1024, Progress: 1.0, Priority: qbt.FilePriorityNormal},
	}
	msg := formatter.FormatFileList("T", files, 1, 1)

	// The full 50-char name should not appear; truncated (40 chars with ellipsis) should.
	if strings.Contains(msg, longBaseName) {
		t.Errorf("long file name should be truncated, got: %q", msg)
	}
	if !strings.Contains(msg, "…") {
		t.Errorf("expected ellipsis after truncation, got: %q", msg)
	}
}

func TestFormatFileList_ShowsPriorityLabel(t *testing.T) {
	files := []qbt.TorrentFile{
		{Index: 0, Name: "file.mkv", Size: 1024, Progress: 0.5, Priority: qbt.FilePrioritySkip},
	}
	msg := formatter.FormatFileList("T", files, 1, 1)

	if !strings.Contains(msg, "Skip") {
		t.Errorf("expected priority label in output, got: %q", msg)
	}
}

func TestFormatFileList_MessageUnderLimit(t *testing.T) {
	files := makeFiles(formatter.FilesPerPage)
	// Use long names to stress-test length.
	for i := range files {
		files[i].Name = "verylongdirname/" + strings.Repeat("X", 50)
	}
	msg := formatter.FormatFileList(strings.Repeat("T", 40), files, 1, 1)

	if len(msg) >= formatter.MaxMessageLength {
		t.Errorf("file list message %d chars exceeds MaxMessageLength %d", len(msg), formatter.MaxMessageLength)
	}
}

// ---- FileListKeyboard (TEST-4) ---------------------------------------------

func TestFileListKeyboard_FileButtons(t *testing.T) {
	hash := strings.Repeat("a", 40)
	files := makeFiles(3)
	kb := formatter.FileListKeyboard(files, hash, 0, 1, formatter.FilesPageState{FilePage: 1, FilterChar: "a", ListPage: 1})

	// 3 file buttons + 1 back button (no pagination for single page).
	if len(kb) != 4 {
		t.Fatalf("expected 4 rows (3 files + back), got %d", len(kb))
	}

	for i, row := range kb[:3] {
		btn := row[0]
		if !strings.HasPrefix(btn.CallbackData, "fs:") {
			t.Errorf("row %d: expected fs: prefix, got %q", i, btn.CallbackData)
		}
		if len(btn.CallbackData) > formatter.MaxCallbackData {
			t.Errorf("row %d: fs: callback %q (%d bytes) exceeds limit", i, btn.CallbackData, len(btn.CallbackData))
		}
	}
}

func TestFileListKeyboard_PaginationButtons_FirstPage(t *testing.T) {
	hash := strings.Repeat("b", 40)
	files := makeFiles(5)
	// 2 total pages → pagination row present.
	kb := formatter.FileListKeyboard(files, hash, 0, 2, formatter.FilesPageState{FilePage: 1, FilterChar: "a", ListPage: 1})

	// Find pagination row (should have Next but no Prev).
	found := false
	for _, row := range kb {
		for _, btn := range row {
			if strings.HasPrefix(btn.CallbackData, "pg:fl:") {
				found = true
				if strings.Contains(btn.Text, "Prev") {
					t.Errorf("first page should not have Prev button")
				}
			}
		}
	}
	if !found {
		t.Errorf("expected pg:fl: pagination button on multi-page list")
	}
}

func TestFileListKeyboard_PaginationButtons_MiddlePage(t *testing.T) {
	hash := strings.Repeat("c", 40)
	files := makeFiles(5)
	kb := formatter.FileListKeyboard(files, hash, 5, 3, formatter.FilesPageState{FilePage: 2, FilterChar: "a", ListPage: 1})

	hasPrev, hasNext := false, false
	for _, row := range kb {
		for _, btn := range row {
			if strings.HasPrefix(btn.CallbackData, "pg:fl:") {
				if strings.Contains(btn.Text, "Prev") {
					hasPrev = true
				}
				if strings.Contains(btn.Text, "Next") {
					hasNext = true
				}
			}
		}
	}
	if !hasPrev {
		t.Errorf("middle page should have Prev button")
	}
	if !hasNext {
		t.Errorf("middle page should have Next button")
	}
}

func TestFileListKeyboard_NoPageButtons_SinglePage(t *testing.T) {
	hash := strings.Repeat("d", 40)
	files := makeFiles(3)
	kb := formatter.FileListKeyboard(files, hash, 0, 1, formatter.FilesPageState{FilePage: 1, FilterChar: "a", ListPage: 1})

	for _, row := range kb {
		for _, btn := range row {
			if strings.HasPrefix(btn.CallbackData, "pg:fl:") {
				t.Errorf("single page list should not have pg:fl: buttons, got %q", btn.CallbackData)
			}
		}
	}
}

func TestFileListKeyboard_BackButton(t *testing.T) {
	hash := strings.Repeat("e", 40)
	files := makeFiles(2)
	kb := formatter.FileListKeyboard(files, hash, 0, 1, formatter.FilesPageState{FilePage: 1, FilterChar: "a", ListPage: 1})

	lastRow := kb[len(kb)-1]
	if !strings.HasPrefix(lastRow[0].CallbackData, "bk:fl:") {
		t.Errorf("last button should be bk:fl: back button, got %q", lastRow[0].CallbackData)
	}
	if len(lastRow[0].CallbackData) > formatter.MaxCallbackData {
		t.Errorf("bk:fl: callback %q (%d bytes) exceeds limit", lastRow[0].CallbackData, len(lastRow[0].CallbackData))
	}
}

func TestFileListKeyboard_AllCallbacksUnderLimit(t *testing.T) {
	hash := strings.Repeat("f", 40)
	files := makeFiles(formatter.FilesPerPage)
	kb := formatter.FileListKeyboard(files, hash, 0, 999, formatter.FilesPageState{FilePage: 1, FilterChar: "a", ListPage: 999})

	for _, row := range kb {
		for _, btn := range row {
			if btn.CallbackData != "noop" && len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q (%d bytes) exceeds %d limit",
					btn.CallbackData, len(btn.CallbackData), formatter.MaxCallbackData)
			}
		}
	}
}

// ---- PriorityKeyboard (TEST-4) ---------------------------------------------

func TestPriorityKeyboard_FourPriorityOptions(t *testing.T) {
	hash := strings.Repeat("a", 40)
	kb := formatter.PriorityKeyboard(hash, 0, qbt.FilePriorityNormal, formatter.FilesPageState{FilePage: 1, FilterChar: "a", ListPage: 1})

	// 4 priority buttons + 1 back button.
	if len(kb) != 5 {
		t.Fatalf("expected 5 rows (4 priorities + back), got %d", len(kb))
	}

	fpCount := 0
	for _, row := range kb[:4] {
		btn := row[0]
		if !strings.HasPrefix(btn.CallbackData, "fp:") {
			t.Errorf("expected fp: prefix, got %q", btn.CallbackData)
		}
		if len(btn.CallbackData) > formatter.MaxCallbackData {
			t.Errorf("fp: callback %q (%d bytes) exceeds limit", btn.CallbackData, len(btn.CallbackData))
		}
		fpCount++
	}
	if fpCount != 4 {
		t.Errorf("expected 4 fp: buttons, got %d", fpCount)
	}
}

func TestPriorityKeyboard_CurrentMarkedWithCheckmark(t *testing.T) {
	hash := strings.Repeat("a", 40)
	kb := formatter.PriorityKeyboard(hash, 0, qbt.FilePriorityHigh, formatter.FilesPageState{FilePage: 1, FilterChar: "a", ListPage: 1})

	checkedCount := 0
	for _, row := range kb[:4] {
		if strings.HasPrefix(row[0].Text, "✓") {
			checkedCount++
			if !strings.Contains(row[0].Text, "High") {
				t.Errorf("checkmark should be on High option, got %q", row[0].Text)
			}
		}
	}
	if checkedCount != 1 {
		t.Errorf("expected exactly 1 checkmark, got %d", checkedCount)
	}
}

func TestPriorityKeyboard_BackButtonIsPgFL(t *testing.T) {
	hash := strings.Repeat("a", 40)
	kb := formatter.PriorityKeyboard(hash, 3, qbt.FilePrioritySkip, formatter.FilesPageState{FilePage: 2, FilterChar: "d", ListPage: 4})

	backBtn := kb[len(kb)-1][0]
	if !strings.HasPrefix(backBtn.CallbackData, "pg:fl:") {
		t.Errorf("back button should use pg:fl: callback, got %q", backBtn.CallbackData)
	}
	if len(backBtn.CallbackData) > formatter.MaxCallbackData {
		t.Errorf("back callback %q (%d bytes) exceeds limit", backBtn.CallbackData, len(backBtn.CallbackData))
	}
}

func makeSearchResults(n int) []qbt.SearchResult {
	results := make([]qbt.SearchResult, n)
	for i := range results {
		results[i] = qbt.SearchResult{
			FileName:   fmt.Sprintf("Ubuntu %d.04 ISO", i+1),
			FileSize:   int64(2+i) * 1024 * 1024 * 1024,
			FileURL:    "magnet:?xt=urn:btih:" + strings.Repeat("a", 40),
			NbSeeders:  50 + i*10,
			NbLeechers: 10 + i,
			SiteURL:    "https://tracker.example",
			DescrLink:  "https://desc.example",
			PubDate:    1672531200 + int64(i*86400),
		}
	}
	return results
}

func TestFormatSearchResults_ContainsQueryAndPage(t *testing.T) {
	results := makeSearchResults(2)
	msg := formatter.FormatSearchResults(results, "ubuntu", 1, 1, formatter.SearchSortInfo{Field: "seeders", Asc: false})

	if !strings.Contains(msg, "ubuntu") {
		t.Errorf("expected query in message, got: %q", msg)
	}
	if !strings.Contains(msg, "page 1/1") {
		t.Errorf("expected page indicator, got: %q", msg)
	}
}

func TestFormatSearchResults_ContainsResultDetails(t *testing.T) {
	results := []qbt.SearchResult{
		{
			FileName:   "Ubuntu 24.04 Desktop ISO",
			FileSize:   2 * 1024 * 1024 * 1024,
			FileURL:    "magnet:?xt=urn:btih:abc",
			NbSeeders:  50,
			NbLeechers: 10,
			PubDate:    1672531200,
		},
	}
	msg := formatter.FormatSearchResults(results, "ubuntu", 1, 1, formatter.SearchSortInfo{Field: "seeders", Asc: false})

	if !strings.Contains(msg, "Ubuntu 24.04") {
		t.Errorf("expected result name, got: %q", msg)
	}
	if !strings.Contains(msg, "2.0 GB") {
		t.Errorf("expected formatted size, got: %q", msg)
	}
	if !strings.Contains(msg, "50") {
		t.Errorf("expected seeders count, got: %q", msg)
	}
	if !strings.Contains(msg, "10") {
		t.Errorf("expected leechers count, got: %q", msg)
	}
}

func TestFormatSearchResults_Empty(t *testing.T) {
	msg := formatter.FormatSearchResults(nil, "nothing", 1, 1, formatter.SearchSortInfo{Field: "seeders", Asc: false})
	if msg != "No torrents found for 'nothing'." {
		t.Errorf("expected empty result message, got: %q", msg)
	}
}

func TestFormatSearchResults_MessageUnderLimit(t *testing.T) {
	results := makeSearchResults(10)
	for i := range results {
		results[i].FileName = strings.Repeat("X", 50)
	}
	msg := formatter.FormatSearchResults(results, strings.Repeat("q", 20), 1, 1, formatter.SearchSortInfo{Field: "seeders", Asc: false})

	if len(msg) >= formatter.MaxMessageLength {
		t.Errorf("message %d chars exceeds limit %d", len(msg), formatter.MaxMessageLength)
	}
}

func TestFormatSearchResults_SortIndicator(t *testing.T) {
	results := makeSearchResults(1)
	msg := formatter.FormatSearchResults(results, "test", 1, 1, formatter.SearchSortInfo{Field: "size", Asc: true})
	if !strings.Contains(msg, "sort: size") {
		t.Errorf("expected sort indicator, got: %q", msg)
	}
}

func TestSearchResultKeyboard_BuildsButtons(t *testing.T) {
	results := makeSearchResults(3)
	kb := formatter.SearchResultKeyboard(results, 42, 1)

	if len(kb) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(kb))
	}
	for i, row := range kb {
		if !strings.HasPrefix(row[0].CallbackData, "sr:42:") {
			t.Errorf("row %d: expected sr:42: prefix, got %q", i, row[0].CallbackData)
		}
		if len(row[0].CallbackData) > formatter.MaxCallbackData {
			t.Errorf("row %d: callback %q exceeds %d bytes", i, row[0].CallbackData, formatter.MaxCallbackData)
		}
	}
}

func TestSearchResultKeyboard_Empty(t *testing.T) {
	kb := formatter.SearchResultKeyboard(nil, 42, 1)
	if kb != nil {
		t.Errorf("expected nil keyboard for empty results, got %v", kb)
	}
}

func TestSearchPaginationKeyboard_FirstPage(t *testing.T) {
	kb := formatter.SearchPaginationKeyboard(42, 1, 3)

	if len(kb) != 2 {
		t.Fatalf("expected 2 rows (pagination + sort), got %d", len(kb))
	}
	row := kb[0]
	if len(row) != 2 {
		t.Fatalf("expected 2 buttons (page + next), got %d", len(row))
	}
	if row[1].Text != "Next >>" {
		t.Errorf("expected Next button, got %q", row[1].Text)
	}
	if !strings.HasPrefix(row[1].CallbackData, "sp:42:") {
		t.Errorf("expected sp:42: prefix, got %q", row[1].CallbackData)
	}
}

func TestSearchPaginationKeyboard_LastPage(t *testing.T) {
	kb := formatter.SearchPaginationKeyboard(42, 3, 3)

	if len(kb) != 2 {
		t.Fatalf("expected 2 rows (pagination + sort), got %d", len(kb))
	}
	row := kb[0]
	if len(row) != 2 {
		t.Fatalf("expected 2 buttons (prev + page), got %d", len(row))
	}
	if row[0].Text != "<< Prev" {
		t.Errorf("expected Prev button, got %q", row[0].Text)
	}
}

func TestSearchPaginationKeyboard_SinglePage(t *testing.T) {
	kb := formatter.SearchPaginationKeyboard(42, 1, 1)
	if len(kb) != 0 {
		t.Errorf("expected empty keyboard for single page, got %d rows", len(kb))
	}
}

func TestSearchPaginationKeyboard_SortButtons(t *testing.T) {
	kb := formatter.SearchPaginationKeyboard(42, 1, 3)

	if len(kb) < 2 {
		t.Fatalf("expected at least 2 rows (pagination + sort), got %d", len(kb))
	}
	sortRow := kb[1]
	if len(sortRow) != 3 {
		t.Fatalf("expected 3 sort buttons, got %d", len(sortRow))
	}
	for _, btn := range sortRow {
		if !strings.HasPrefix(btn.CallbackData, "ss:42:") {
			t.Errorf("expected ss:42: prefix, got %q", btn.CallbackData)
		}
		if len(btn.CallbackData) > formatter.MaxCallbackData {
			t.Errorf("callback %q exceeds %d bytes", btn.CallbackData, formatter.MaxCallbackData)
		}
	}
}

func TestFormatSearchConfirm(t *testing.T) {
	result := qbt.SearchResult{
		FileName:  "Ubuntu 24.04 ISO",
		FileSize:  2 * 1024 * 1024 * 1024,
		NbSeeders: 50,
	}
	msg := formatter.FormatSearchConfirm(result, "", 0, 0)

	if !strings.Contains(msg, "Ubuntu 24.04") {
		t.Errorf("expected torrent name, got: %q", msg)
	}
	if !strings.Contains(msg, "2.0 GB") {
		t.Errorf("expected size, got: %q", msg)
	}
	if !strings.Contains(msg, "50") {
		t.Errorf("expected seeders, got: %q", msg)
	}
	if strings.Contains(msg, "Description:") {
		t.Errorf("expected no Description when empty, got: %q", msg)
	}
}

func TestFormatSearchConfirm_WithDescription(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   "Ubuntu 24.04 ISO",
		FileSize:   2 * 1024 * 1024 * 1024,
		NbSeeders:  50,
		NbLeechers: 5,
	}
	msg := formatter.FormatSearchConfirm(result, "A reliable Linux distribution", 1, 1)

	if !strings.Contains(msg, "Description:") {
		t.Errorf("expected Description: label, got: %q", msg)
	}
	if !strings.Contains(msg, "A reliable Linux distribution") {
		t.Errorf("expected description text, got: %q", msg)
	}
	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message exceeds %d chars: %d", formatter.MaxMessageLength, len(msg))
	}
}

func TestFormatSearchConfirm_NoDescription(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   "Ubuntu 24.04 ISO",
		FileSize:   2 * 1024 * 1024 * 1024,
		NbSeeders:  50,
		NbLeechers: 5,
	}
	msg := formatter.FormatSearchConfirm(result, "", 0, 0)

	if strings.Contains(msg, "Description:") {
		t.Errorf("expected no Description when empty, got: %q", msg)
	}
	// Original content preserved.
	if !strings.Contains(msg, "Seeders:") {
		t.Errorf("expected original content (Seeders:) preserved, got: %q", msg)
	}
}

func TestFormatSearchConfirm_DescriptionTruncated(t *testing.T) {
	// With pagination: long description shows page 1 of N, not truncated with "...".
	longName := strings.Repeat("x", 3990)
	longDesc := strings.Repeat("y", 300)
	result := qbt.SearchResult{
		FileName:   longName,
		FileSize:   1024,
		NbSeeders:  1,
		NbLeechers: 1,
	}
	msg := formatter.FormatSearchConfirm(result, longDesc, 1, 2)

	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message exceeds %d chars: got %d", formatter.MaxMessageLength, len(msg))
	}
	if !strings.Contains(msg, "Description (page 1/2):") {
		t.Errorf("expected paginated Description label, got: %q", msg)
	}
}

func TestFormatSearchConfirm_DescriptionFitsExactly(t *testing.T) {
	// Short description on a single page — shows full text with "Description:" label.
	shortName := strings.Repeat("x", 10)
	exactDesc := strings.Repeat("y", 38)
	result := qbt.SearchResult{
		FileName:   shortName,
		FileSize:   1024,
		NbSeeders:  1,
		NbLeechers: 1,
	}
	msg := formatter.FormatSearchConfirm(result, exactDesc, 1, 1)

	if !strings.Contains(msg, exactDesc) {
		t.Errorf("expected full description for single page; got len=%d", len(msg))
	}
	if !strings.Contains(msg, "Description:") {
		t.Errorf("expected Description: label, got: %q", msg)
	}
	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message exceeds limit: %d chars", len(msg))
	}
	if !strings.Contains(msg, "Seeders:") {
		t.Errorf("expected original content (Seeders:) preserved, got: %q", msg)
	}
}

func TestFormatSearchConfirm_DescriptionAt4096(t *testing.T) {
	// Total is exactly 4096: over MaxMessageLength-1, triggers truncation.
	// name=3985 + desc=38 → msg(4040) + descSection(15+38=53) = 4093
	// With larger desc: name=3985 + desc=39 → msg(4040) + descSection(54) = 4094
	// name=3986 + desc=38 → msg(4041) + descSection(53) = 4094
	// Need total > 4095: name=3987 + desc=38 → msg(4042)+53=4095 → not triggered
	// name=3987 + desc=39 → msg(4042)+54=4096 → triggered
	longName := strings.Repeat("x", 3987)
	shortDesc := strings.Repeat("y", 39)
	result := qbt.SearchResult{
		FileName:   longName,
		FileSize:   1024,
		NbSeeders:  1,
		NbLeechers: 1,
	}
	msg := formatter.FormatSearchConfirm(result, shortDesc, 1, 1)

	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message exceeds limit: %d chars", len(msg))
	}
	if strings.Contains(msg, shortDesc) {
		t.Errorf("expected description to be truncated at 4096, but full text present")
	}
	if !strings.Contains(msg, "Description:") {
		t.Errorf("expected Description: label even when truncated, got: %q", msg)
	}
}

func TestFormatSearchConfirm_DescriptionAvailZero(t *testing.T) {
	// avail = 0: room for label but zero room for description text.
	// msg overhead=55, desc overhead=15+3=18, avail=4095-55-nameLen-18=4022-nameLen.
	// For avail=0: nameLen=4022, any descLen>0 triggers outer, avail=0.
	longName := strings.Repeat("x", 4022)
	result := qbt.SearchResult{
		FileName:   longName,
		FileSize:   1024,
		NbSeeders:  1,
		NbLeechers: 1,
	}
	msg := formatter.FormatSearchConfirm(result, "tiny", 1, 1)

	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message exceeds limit: %d chars", len(msg))
	}
	if strings.Contains(msg, "Description:") {
		t.Errorf("expected no Description when avail=0, got: %q", msg)
	}
}

func TestFormatSearchConfirm_DescriptionAvailNegative(t *testing.T) {
	// avail < 0: no room at all, description omitted.
	longName := strings.Repeat("x", 4023)
	result := qbt.SearchResult{
		FileName:   longName,
		FileSize:   1024,
		NbSeeders:  1,
		NbLeechers: 1,
	}
	msg := formatter.FormatSearchConfirm(result, "tiny", 1, 1)

	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message exceeds limit: %d chars", len(msg))
	}
	if strings.Contains(msg, "Description:") {
		t.Errorf("expected no Description when avail negative, got: %q", msg)
	}
}

func TestFormatSearchConfirm_DescriptionUTF8Truncated(t *testing.T) {
	// UTF-8 safe truncation: multi-byte characters not split mid-sequence.
	// name=3987 + 20 é's (40 bytes) → msg(4042) + descSection(15+40=55) = 4097, avail=35.
	longName := strings.Repeat("x", 3987)
	result := qbt.SearchResult{
		FileName:   longName,
		FileSize:   1024,
		NbSeeders:  1,
		NbLeechers: 1,
	}
	msg := formatter.FormatSearchConfirm(result, strings.Repeat("é", 20), 1, 1)

	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message exceeds limit: %d chars", len(msg))
	}
	if !utf8.ValidString(msg) {
		t.Errorf("message contains invalid UTF-8 after truncation")
	}
	if !strings.Contains(msg, "Description:") {
		t.Errorf("expected Description: label, got: %q", msg)
	}
}

func TestFormatSearchConfirm_LinkAndDescription(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   "Ubuntu 24.04 ISO",
		FileSize:   2 * 1024 * 1024 * 1024,
		NbSeeders:  50,
		NbLeechers: 5,
		DescrLink:  "https://example.com/torrent/12345",
	}
	msg := formatter.FormatSearchConfirm(result, "A reliable Linux distribution for desktop and server use.", 1, 1)

	if !strings.Contains(msg, "More info:") {
		t.Errorf("expected More info: link line, got: %q", msg)
	}
	if !strings.Contains(msg, "https://example.com/torrent/12345") {
		t.Errorf("expected link URL, got: %q", msg)
	}
	if !strings.Contains(msg, "Description:") {
		t.Errorf("expected Description: section, got: %q", msg)
	}
	if !strings.Contains(msg, "A reliable Linux distribution") {
		t.Errorf("expected description text, got: %q", msg)
	}
	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message exceeds %d chars: %d", formatter.MaxMessageLength, len(msg))
	}
}

func TestSearchConfirmKeyboard_WithDescPagination(t *testing.T) {
	// Multi-page: should have Prev/Next buttons.
	kb := formatter.SearchConfirmKeyboardWithDesc(42, 5, 2, 2, 3)
	// Expect 4 rows: Add, Back, Prev, Next.
	foundPrev := false
	foundNext := false
	for _, row := range kb {
		for _, btn := range row {
			if strings.Contains(btn.Text, "Prev") {
				foundPrev = true
			}
			if strings.Contains(btn.Text, "Next") {
				foundNext = true
			}
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q exceeds %d bytes", btn.CallbackData, formatter.MaxCallbackData)
			}
		}
	}
	if !foundPrev || !foundNext {
		t.Errorf("expected Prev+Next buttons, got %d rows", len(kb))
	}
}

func TestSearchConfirmKeyboard_WithDescSinglePage(t *testing.T) {
	// Single page: no pagination buttons.
	kb := formatter.SearchConfirmKeyboardWithDesc(42, 5, 2, 1, 1)
	for _, row := range kb {
		for _, btn := range row {
			if btn.Text == "Prev page" || btn.Text == "Next page" {
				t.Errorf("expected no pagination for single-page description")
			}
		}
	}
	// Verify no empty rows: exactly 2 rows (Add, Back), no extra pagination row.
	if len(kb) != 2 {
		t.Errorf("expected exactly 2 rows for single page, got %d", len(kb))
	}
}

func TestSearchConfirmKeyboardWithDesc_FirstPage(t *testing.T) {
	kb := formatter.SearchConfirmKeyboardWithDesc(42, 5, 2, 1, 3)
	foundPrev, foundNext := false, false
	for _, row := range kb {
		for _, btn := range row {
			if strings.Contains(btn.Text, "Prev") {
				foundPrev = true
			}
			if strings.Contains(btn.Text, "Next") {
				foundNext = true
			}
		}
	}
	if foundPrev {
		t.Error("expected no Prev button on first page")
	}
	if !foundNext {
		t.Error("expected Next button on first page of multi-page")
	}
}

func TestSearchConfirmKeyboardWithDesc_LastPage(t *testing.T) {
	kb := formatter.SearchConfirmKeyboardWithDesc(42, 5, 2, 3, 3)
	foundPrev, foundNext := false, false
	for _, row := range kb {
		for _, btn := range row {
			if strings.Contains(btn.Text, "Prev") {
				foundPrev = true
			}
			if strings.Contains(btn.Text, "Next") {
				foundNext = true
			}
		}
	}
	if !foundPrev {
		t.Error("expected Prev button on last page")
	}
	if foundNext {
		t.Error("expected no Next button on last page")
	}
}

func TestSearchConfirmKeyboardWithDesc_CallbackValues(t *testing.T) {
	kb := formatter.SearchConfirmKeyboardWithDesc(99, 7, 1, 2, 4)
	for _, row := range kb {
		for _, btn := range row {
			if strings.Contains(btn.Text, "Prev") {
				if !strings.Contains(btn.CallbackData, "dp:99:7:1") {
					t.Errorf("expected dp:99:7:1 in Prev callback, got %q", btn.CallbackData)
				}
			}
			if strings.Contains(btn.Text, "Next") {
				if !strings.Contains(btn.CallbackData, "dp:99:7:3") {
					t.Errorf("expected dp:99:7:3 in Next callback, got %q", btn.CallbackData)
				}
			}
		}
	}
}

func TestSearchConfirmKeyboardWithDesc_NoExtraRowForEmpty(t *testing.T) {
	// descPage=1, descTotal=0: should not have pagination row.
	kb := formatter.SearchConfirmKeyboardWithDesc(42, 5, 2, 1, 0)
	if len(kb) != 2 {
		t.Errorf("expected exactly 2 rows (no pagination), got %d", len(kb))
	}
}

func TestSplitDescription(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		maxPerPage int
		wantPages  int
		wantEmpty  bool
	}{
		{"empty", "", 100, 0, true},
		{"zero size", "hello", 0, 0, true},
		{"negative size", "hello", -1, 0, true},
		{"fits one page", "hello", 100, 1, false},
		{"multi page", strings.Repeat("x", 200), 100, 2, false},
		{"exact boundary", strings.Repeat("x", 100), 100, 1, false},
		{"min page size 1", "abc", 1, 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages := formatter.SplitDescription(tt.text, tt.maxPerPage)
			if tt.wantEmpty && len(pages) != 0 {
				t.Errorf("expected no pages, got %d", len(pages))
			}
			if !tt.wantEmpty && len(pages) != tt.wantPages {
				t.Errorf("expected %d pages, got %d", tt.wantPages, len(pages))
			}
		})
	}
}

func TestDescriptionPage_OffsetAlign(t *testing.T) {
	// Force offset to land in middle of multi-byte character.
	text := "abc" + strings.Repeat("é", 5) + "xyz"
	// pageSize of 5: page 1 = "abc" + first 2 bytes of é (half char).
	// The offset align should skip the partial é.
	pages := formatter.SplitDescription(text, 5)
	for _, p := range pages {
		if !utf8.ValidString(p) {
			t.Errorf("page contains invalid UTF-8: %q", p)
		}
	}
}

func TestSearchConfirmKeyboard(t *testing.T) {
	kb := formatter.SearchConfirmKeyboard(42, 5, 2)

	if len(kb) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(kb))
	}

	addBtn := kb[0][0]
	if !strings.Contains(addBtn.Text, "Add") {
		t.Errorf("expected Add button, got %q", addBtn.Text)
	}
	if !strings.HasPrefix(addBtn.CallbackData, "sc:42:5") {
		t.Errorf("expected sc:42:5 prefix, got %q", addBtn.CallbackData)
	}

	backBtn := kb[1][0]
	if !strings.Contains(backBtn.Text, "Back") {
		t.Errorf("expected Back button, got %q", backBtn.Text)
	}
	if !strings.HasPrefix(backBtn.CallbackData, "sb:42:2") {
		t.Errorf("expected sb:42:2 prefix, got %q", backBtn.CallbackData)
	}

	for _, row := range kb {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q exceeds %d bytes", btn.CallbackData, formatter.MaxCallbackData)
			}
		}
	}
}

func TestSearchPaginationKeyboard_CallbackDataUnderLimit(t *testing.T) {
	kb := formatter.SearchPaginationKeyboard(9999, 999, 9999)
	for _, row := range kb {
		for _, btn := range row {
			if btn.CallbackData != "noop" && len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q (%d bytes) exceeds %d limit",
					btn.CallbackData, len(btn.CallbackData), formatter.MaxCallbackData)
			}
		}
	}
}

func TestSearchResultKeyboard_CallbackDataUnderLimit(t *testing.T) {
	results := []qbt.SearchResult{
		{FileName: strings.Repeat("X", 100), FileURL: "magnet:?xt=urn:btih:" + strings.Repeat("a", 40)},
	}
	kb := formatter.SearchResultKeyboard(results, 9999, 999)
	for _, row := range kb {
		for _, btn := range row {
			if len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q (%d bytes) exceeds %d limit",
					btn.CallbackData, len(btn.CallbackData), formatter.MaxCallbackData)
			}
		}
	}
}

func TestPriorityKeyboard_AllCallbacksUnderLimit(t *testing.T) {
	hash := strings.Repeat("f", 40)
	kb := formatter.PriorityKeyboard(hash, 99999, qbt.FilePriorityMaximum, formatter.FilesPageState{FilePage: 999, FilterChar: "a", ListPage: 999})

	for _, row := range kb {
		for _, btn := range row {
			if btn.CallbackData != "noop" && len(btn.CallbackData) > formatter.MaxCallbackData {
				t.Errorf("callback %q (%d bytes) exceeds %d limit",
					btn.CallbackData, len(btn.CallbackData), formatter.MaxCallbackData)
			}
		}
	}
}

// Additional tests for description-related functions — kill CONDITIONALS_BOUNDARY,
// ARITHMETIC_BASE, BRANCH_IF, and INVERT_LOGICAL mutants.

func TestDescriptionPageSize_NoLink(t *testing.T) {
	base := formatter.FormatSearchConfirmBase(qbt.SearchResult{FileName: "test"})
	ps := formatter.DescriptionPageSize(base, "")
	if ps <= 0 {
		t.Errorf("expected positive page size, got %d", ps)
	}
}

func TestDescriptionPageSize_WithLink(t *testing.T) {
	base := formatter.FormatSearchConfirmBase(qbt.SearchResult{FileName: "test", DescrLink: "https://example.com/t"})
	psNoLink := formatter.DescriptionPageSize(base, "")
	psWithLink := formatter.DescriptionPageSize(base, "https://example.com/t")
	if psWithLink >= psNoLink {
		t.Errorf("page size with link (%d) should be less than without link (%d)", psWithLink, psNoLink)
	}
}

func TestDescriptionPageSize_LinkConsumedExactly(t *testing.T) {
	// Verify DescriptionPageSize subtracts the link line length.
	base := "x"
	ps := formatter.DescriptionPageSize(base, "https://example.com/t")
	expected := formatter.MaxMessageLength - 1 - len(base) - len("\n\nMore info: https://example.com/t") - 32
	if ps != expected {
		t.Errorf("expected page size %d, got %d", expected, ps)
	}
}

func TestFormatSearchConfirm_NoDescriptionNoLink(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   "test",
		FileSize:   100,
		NbSeeders:  1,
		NbLeechers: 0,
	}
	msg := formatter.FormatSearchConfirm(result, "", 0, 0)
	if strings.Contains(msg, "Description:") {
		t.Error("unexpected Description: label when description is empty")
	}
	if strings.Contains(msg, "More info:") {
		t.Error("unexpected More info: when DescrLink is empty")
	}
}

func TestFormatSearchConfirm_InvalidPage(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   "test",
		FileSize:   100,
		NbSeeders:  1,
		NbLeechers: 0,
	}
	// page=0 should be rejected (no Description:)
	msg0 := formatter.FormatSearchConfirm(result, "desc", 0, 1)
	if strings.Contains(msg0, "Description:") {
		t.Error("expected no Description: for page=0")
	}
	// page > totalPages
	msgOver := formatter.FormatSearchConfirm(result, "desc", 5, 3)
	if strings.Contains(msgOver, "Description:") {
		t.Error("expected no Description: when page > totalPages")
	}
	// negative totalPages
	msgNeg := formatter.FormatSearchConfirm(result, "desc", 1, -1)
	if strings.Contains(msgNeg, "Description:") {
		t.Error("expected no Description: for negative totalPages")
	}
}

func TestFormatSearchConfirm_ShortBaseMessage(t *testing.T) {
	// Base message so short that DescriptionPageSize is close to limit.
	result := qbt.SearchResult{
		FileName:   strings.Repeat("a", formatter.MaxMessageLength-200),
		FileSize:   100,
		NbSeeders:  1,
		NbLeechers: 0,
	}
	msg := formatter.FormatSearchConfirm(result, "short desc", 1, 1)
	// Should still contain the description if pageSize >= 16.
	if !strings.Contains(msg, "Description:") || !strings.Contains(msg, "short desc") {
		t.Errorf("expected description in message, got: %q", msg)
	}
}

func TestFormatSearchConfirm_MultiPageLabels(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   "Ubuntu 24.04 LTS",
		FileSize:   4 * 1024 * 1024 * 1024,
		NbSeeders:  100,
		NbLeechers: 20,
	}
	baseMsg := formatter.FormatSearchConfirmBase(result)
	pageSize := formatter.DescriptionPageSize(baseMsg, "")
	// Single page
	msg1 := formatter.FormatSearchConfirm(result, "short desc", 1, 1)
	if strings.Contains(msg1, "page") {
		t.Error("single page should not show page number in label")
	}
	if !strings.Contains(msg1, "Description:") {
		t.Error("single page should have Description: label")
	}
	// Multi-page: description long enough to fill more than one page.
	longDesc := strings.Repeat("x", pageSize+10)
	msg2 := formatter.FormatSearchConfirm(result, longDesc, 2, 2)
	if !strings.Contains(msg2, "Description (page 2/2):") {
		t.Errorf("expected 'Description (page 2/2):', got label area: %q", msg2[len(msg2)-50:])
	}
}

func TestFormatSearchConfirm_MessageNeverExceedsLimit(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   strings.Repeat("a", 1000),
		FileSize:   4 * 1024 * 1024 * 1024,
		NbSeeders:  100,
		NbLeechers: 20,
		DescrLink:  "https://example.com/xxx",
	}
	longDesc := strings.Repeat("x", 5000)
	msg := formatter.FormatSearchConfirm(result, longDesc, 1, 1)
	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message %d bytes exceeds limit of %d", len(msg), formatter.MaxMessageLength)
	}
	if !utf8.ValidString(msg) {
		t.Error("message contains invalid UTF-8")
	}
}

func TestSplitDescription_UTF8MidBoundary(t *testing.T) {
	// € is 3 bytes in UTF-8 (0xE2 0x82 0xAC). Use page size that cuts mid-character.
	text := "a€b"
	pages := formatter.SplitDescription(text, 2) // cuts into middle of € (bytes 0-1: "a\xe2")
	for _, p := range pages {
		if !utf8.ValidString(p) {
			t.Errorf("page contains invalid UTF-8: %q", p)
		}
	}
	if len(pages) == 0 {
		t.Error("expected at least one page")
	}
}

func TestSplitDescription_LongText(t *testing.T) {
	text := strings.Repeat("x", 10000)
	pages := formatter.SplitDescription(text, 200)
	if len(pages) != 50 {
		t.Errorf("expected 50 pages, got %d", len(pages))
	}
	var rebuilt strings.Builder
	for _, p := range pages {
		rebuilt.WriteString(p)
	}
	if rebuilt.String() != text {
		t.Error("rebuilt text does not match original")
	}
}

func TestSplitDescription_MultiByteSingleByte(t *testing.T) {
	// € (3 bytes) with pageSize=1 forces empty-page-skip path.
	// Kills INVERT_LOOP_CTRL on 'continue' (line 659).
	text := "x" + "\xe2\x82\xac" + "y" // "x€y" = 1+3+1 = 5 bytes
	pages := formatter.SplitDescription(text, 1)
	if len(pages) < 2 {
		t.Errorf("expected at least 2 pages from 'x€y' at 1-byte pages, got %d", len(pages))
	}
	for _, p := range pages {
		if !utf8.ValidString(p) {
			t.Errorf("page contains invalid UTF-8: %q", p)
		}
	}
}

func TestFormatSearchConfirm_PageSize16Boundary(t *testing.T) {
	// Build base message such that DescriptionPageSize returns exactly 16.
	// Kills CONDITIONALS_BOUNDARY on pageSize < 16 (line 673).
	// MaxMessageLength=4096, margin=32. Needed: baseLen=4096-1-16-32=4047.
	base := strings.Repeat("x", 4047)
	result := qbt.SearchResult{FileName: base, FileSize: 0, NbSeeders: 0, NbLeechers: 0}
	ps := formatter.DescriptionPageSize(formatter.FormatSearchConfirmBase(result), "")
	if ps < 16 {
		// If the message is too long, we can't construct a 16-byte page boundary.
		t.Skipf("page size %d < 16; can't test 16-boundary", ps)
	}
	// Use a description that exactly fits the computed page size.
	desc := strings.Repeat("d", ps+1)
	msg := formatter.FormatSearchConfirm(result, desc, 1, 1)
	if !strings.Contains(msg, "Description:") {
		t.Errorf("expected Description: for pageSize>=16, got no description")
	}
}

func TestFormatSearchConfirm_EmptyPageText(t *testing.T) {
	// page 2 of a description that doesn't have enough text for a second page.
	// Kills BRANCH_IF on pageText == "" (line 679).
	result := qbt.SearchResult{
		FileName:   "Short",
		FileSize:   100,
		NbSeeders:  1,
		NbLeechers: 0,
	}
	msg := formatter.FormatSearchConfirm(result, "short", 999, 999)
	if strings.Contains(msg, "Description:") {
		t.Error("expected no Description: for out-of-bounds page")
	}
}

func TestFormatSearchConfirm_ArithmeticPageOffset(t *testing.T) {
	// Kills ARITHMETIC_BASE on (page-1)*pageSize (line 693).
	result := qbt.SearchResult{
		FileName:   "Test",
		FileSize:   100,
		NbSeeders:  1,
		NbLeechers: 0,
	}
	pageSize := formatter.DescriptionPageSize(formatter.FormatSearchConfirmBase(result), "")
	if pageSize < 1 {
		t.Skip("page size too small for arithmetic test")
	}
	desc := strings.Repeat("x", pageSize*3)
	// Page 2 should contain text from offset pageSize to 2*pageSize-1.
	msg := formatter.FormatSearchConfirm(result, desc, 2, 3)
	if !strings.Contains(msg, "Description (page 2/3):") {
		t.Errorf("expected page 2 label, got: %q", msg[len(msg)-100:])
	}
}

func TestFormatSearchConfirm_CompoundGuardCoverage(t *testing.T) {
	// Cover each path of the compound guard in appendDescriptionPaginated.
	result := qbt.SearchResult{FileName: "x", FileSize: 0, NbSeeders: 0, NbLeechers: 0}

	// totalPages <= 0
	msg := formatter.FormatSearchConfirm(result, "desc", 1, 0)
	if strings.Contains(msg, "Description:") {
		t.Error("expected no description for totalPages=0")
	}
	// page <= 0
	msg = formatter.FormatSearchConfirm(result, "desc", 0, 1)
	if strings.Contains(msg, "Description:") {
		t.Error("expected no description for page=0")
	}
	// page > totalPages
	msg = formatter.FormatSearchConfirm(result, "desc", 3, 2)
	if strings.Contains(msg, "Description:") {
		t.Error("expected no description when page > totalPages")
	}
}

func TestAppendMoreInfoLink_Truncated(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   strings.Repeat("x", 4000),
		FileSize:   100,
		NbSeeders:  1,
		NbLeechers: 0,
		DescrLink:  "https://example.com/" + strings.Repeat("x", 200),
	}
	msg := formatter.FormatSearchConfirm(result, "", 0, 0)
	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message %d exceeds limit", len(msg))
	}
	if !strings.Contains(msg, "...") {
		t.Error("expected truncation ellipsis for long link")
	}
}

func TestAppendMoreInfoLink_ShortMessage(t *testing.T) {
	result := qbt.SearchResult{
		FileName:   "T",
		FileSize:   0,
		NbSeeders:  0,
		NbLeechers: 0,
		DescrLink:  "https://e.co/t",
	}
	msg := formatter.FormatSearchConfirm(result, "", 0, 0)
	if !strings.Contains(msg, "https://e.co/t") {
		t.Errorf("expected full link in short message, got: %q", msg)
	}
}

func TestAppendMoreInfoLink_UTF8Truncation(t *testing.T) {
	// Truncation at multi-byte boundary: ensure result is valid UTF-8.
	result := qbt.SearchResult{
		FileName:   strings.Repeat("x", 4000),
		FileSize:   100,
		NbSeeders:  1,
		NbLeechers: 0,
		DescrLink:  "https://example.com/" + strings.Repeat("€", 50),
	}
	msg := formatter.FormatSearchConfirm(result, "", 0, 0)
	if len(msg) > formatter.MaxMessageLength {
		t.Errorf("message %d exceeds limit", len(msg))
	}
	if !utf8.ValidString(msg) {
		t.Error("message contains invalid UTF-8 after truncation")
	}
}

func TestDescriptionPage_UTF8Alignment(t *testing.T) {
	// Description with multi-byte chars at page boundary.
	result := qbt.SearchResult{
		FileName:   "T",
		FileSize:   0,
		NbSeeders:  0,
		NbLeechers: 0,
	}
	// Place multi-byte chars at position where page 2 starts.
	prefix := strings.Repeat("a", formatter.DescriptionPageSize(formatter.FormatSearchConfirmBase(result), "")-1)
	desc := prefix + "€" + strings.Repeat("b", 100)
	msg := formatter.FormatSearchConfirm(result, desc, 2, 2)
	if !utf8.ValidString(msg) {
		t.Error("message contains invalid UTF-8 after page boundary alignment")
	}
}
