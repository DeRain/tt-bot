package bot

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	descriptionFetchTimeout = 5 * time.Second
	descriptionMaxBytes     = 256 * 1024 // 256KB max response
)

var (
	metaDescriptionRe = regexp.MustCompile(`<meta[^>]*name\s*=\s*["']description["'][^>]*content\s*=\s*["']([^"']*)["'][^>]*>`)
	ogDescriptionRe   = regexp.MustCompile(`<meta[^>]*property\s*=\s*["']og:description["'][^>]*content\s*=\s*["']([^"']*)["'][^>]*>`)
	htmlTagRe         = regexp.MustCompile(`<[^>]*>`)
	htmlEntityRe      = regexp.MustCompile(`&[a-z]+;`)
)

// descriptionFetcher fetches description text from a torrent description page URL.
type descriptionFetcher struct {
	client *http.Client
}

func newDescriptionFetcher() *descriptionFetcher {
	return &descriptionFetcher{
		client: &http.Client{Timeout: descriptionFetchTimeout},
	}
}

// fetch retrieves description text from the given descrLink URL.
// Returns empty string on any error — callers should degrade gracefully.
func (f *descriptionFetcher) fetch(ctx context.Context, rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return ""
	}
	if err := isPublicHostname(parsed.Host); err != nil {
		return ""
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	req.Header.Set("User-Agent", "tt-bot/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	limitedReader := io.LimitReader(resp.Body, descriptionMaxBytes)
	body, _ := io.ReadAll(limitedReader)

	htmlStr := string(body)
	desc := extractDescription(htmlStr)
	return cleanDescription(desc)
}

func extractDescription(htmlStr string) string {
	if m := metaDescriptionRe.FindStringSubmatch(htmlStr); len(m) >= 2 {
		return decodeEntities(m[1])
	}
	if m := ogDescriptionRe.FindStringSubmatch(htmlStr); len(m) >= 2 {
		return decodeEntities(m[1])
	}
	return extractBodyText(htmlStr)
}

func extractBodyText(htmlStr string) string {
	text := htmlTagRe.ReplaceAllString(htmlStr, " ")
	text = decodeEntities(text)

	var buf strings.Builder
	inSpace := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if !inSpace {
				buf.WriteRune(' ')
				inSpace = true
			}
		} else {
			buf.WriteRune(r)
			inSpace = false
		}
	}
	return strings.TrimSpace(buf.String())
}

func cleanDescription(desc string) string {
	return strings.TrimSpace(desc)
}

var htmlEntities = map[string]string{
	"&amp;":  "&",
	"&lt;":   "<",
	"&gt;":   ">",
	"&quot;": "\"",
	"&#39;":  "'",
	"&nbsp;": " ",
}

func decodeEntities(s string) string {
	s = htmlEntityRe.ReplaceAllStringFunc(s, func(m string) string {
		if repl, ok := htmlEntities[m]; ok {
			return repl
		}
		return m
	})
	return s
}
