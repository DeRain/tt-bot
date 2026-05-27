package formatter

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDescriptionPage_AlignmentPastEnd(t *testing.T) {
	// Page 2 starts at offset that walks alignment to exactly len(text).
	// This kills CONDITIONALS_BOUNDARY on start < len(text) in for-loop
	// because the mutant (<=) would panic at text[len(text)].
	text := "a" + string([]byte{0xe4, 0xb8, 0x96}) // "a世" = 4 bytes
	result := descriptionPage(text, 2, 3)          // page 2, pageSize=3 → start=3, middle of 世
	if utf8.ValidString(result) {
		// Expected: start walks to len(text)=4, guard returns "".
		// Mutant (<=): panics at text[4] → test fails, mutant killed.
		if result != "" {
			t.Errorf("expected empty result when alignment walks past end, got %q", result)
		}
	}
}

func TestDescriptionPage_StartAtEnd(t *testing.T) {
	// Page starts exactly at len(text) — kills CONDITIONALS_BOUNDARY on >=.
	text := "abc"
	result := descriptionPage(text, 2, 3) // page 2, pageSize=3 → start=3 == len(text)
	if result != "" {
		t.Errorf("expected empty result when start == len(text), got %q", result)
	}
}

func TestAppendMoreInfoLink_EndAtZero(t *testing.T) {
	// Base message so long that linkMax == 0 — kills CONDITIONALS_BOUNDARY on end <= 0.
	msg := strings.Repeat("x", MaxMessageLength-len("\n\nMore info: ")-3)
	result := appendMoreInfoLink(msg, "https://example.com/page")
	if strings.Contains(result, "...") {
		t.Error("expected no ellipsis when end == 0 (no room)")
	}
}

func TestAppendDescriptionPaginated_PageSizeAtZero(t *testing.T) {
	// Fill message to exactly MaxMessageLength - descLineOverhead - 32 = 0 pageSize.
	// DescriptionPageSize = MaxMessageLength - 1 - len(msg) - 32 = 0 → len(msg) = MaxMessageLength - 33.
	msg := strings.Repeat("x", MaxMessageLength-33)
	result := appendDescriptionPaginated(msg, "desc", 1, 1)
	if strings.Contains(result, "Description:") {
		t.Error("expected no description when pageSize == 0")
	}
}
