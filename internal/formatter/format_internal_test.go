package formatter

import (
	"strings"
	"testing"
)

func TestAppendMoreInfoLink_EndAtZero(t *testing.T) {
	// Base message so long that linkMax == 0 — kills CONDITIONALS_BOUNDARY on end <= 0.
	msg := strings.Repeat("x", MaxMessageLength-len("\n\nMore info: ")-3)
	result := appendMoreInfoLink(msg, "https://example.com/page")
	if strings.Contains(result, "...") {
		t.Error("expected no ellipsis when end == 0 (no room)")
	}
}
