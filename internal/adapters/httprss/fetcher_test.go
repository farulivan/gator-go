package httprss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFetcher_Fetch_HappyPath verifies the full pipeline on a recorded
// fixture: HTTP GET → XML decode → html.UnescapeString → pubDate parsing.
// The fixture deliberately exercises both unescape and the zero-pubDate
// fallback so this one test pins all three classes of behavior.
func TestFetcher_Fetch_HappyPath(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "example.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var gotUserAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	feed, err := New().Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if gotUserAgent != "gator" {
		t.Errorf("User-Agent = %q, want %q", gotUserAgent, "gator")
	}
	if feed.Title != "Example Blog & News" {
		t.Errorf("Title = %q, want HTML-unescaped 'Example Blog & News'", feed.Title)
	}
	if feed.Description != "Posts from the Example team & friends" {
		t.Errorf("Description = %q, want HTML-unescaped string", feed.Description)
	}

	if len(feed.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(feed.Items))
	}

	want0 := "First Post: Hello & Welcome"
	if feed.Items[0].Title != want0 {
		t.Errorf("Items[0].Title = %q, want %q", feed.Items[0].Title, want0)
	}
	if !strings.Contains(feed.Items[0].Description, "<p>Welcome to the blog!</p>") {
		t.Errorf("Items[0].Description = %q, want HTML entities decoded", feed.Items[0].Description)
	}
	wantPub0 := time.Date(2026, 5, 11, 9, 30, 0, 0, time.UTC)
	if !feed.Items[0].PublishedAt.Equal(wantPub0) {
		t.Errorf("Items[0].PublishedAt = %v, want %v", feed.Items[0].PublishedAt, wantPub0)
	}

	wantPub1 := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	if !feed.Items[1].PublishedAt.Equal(wantPub1) {
		t.Errorf("Items[1].PublishedAt = %v, want %v", feed.Items[1].PublishedAt, wantPub1)
	}

	if !feed.Items[2].PublishedAt.IsZero() {
		t.Errorf("Items[2].PublishedAt = %v, want zero (unparseable date)", feed.Items[2].PublishedAt)
	}
}

// TestFetcher_Fetch_HTTPErrorBody covers the case where the server
// responds with a non-XML body (e.g. an HTML error page on 500). The
// decoder rejects it and the error is surfaced unwrapped.
func TestFetcher_Fetch_InvalidXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not xml at all"))
	}))
	defer srv.Close()

	if _, err := New().Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error on invalid XML body, got nil")
	}
}

// TestFetcher_Fetch_ContextCancelled confirms that a cancelled context
// short-circuits the HTTP call rather than waiting on the server.
func TestFetcher_Fetch_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(5 * time.Second) // would block past the test timeout
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := New().Fetch(ctx, srv.URL); err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}
