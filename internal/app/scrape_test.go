package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/google/uuid"
)

func TestScrapeService_Scrape(t *testing.T) {
	now := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	feed := domain.Feed{
		ID:   uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Name: "Example",
		URL:  "https://example.com/rss",
	}
	pub := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	postIDs := []uuid.UUID{
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		uuid.MustParse("44444444-4444-4444-4444-444444444444"),
	}

	t.Run("happy path classifies items into Inserted/Skipped/Dups", func(t *testing.T) {
		store := &fakeScrapeStore{
			nextFeed: feed,
			createPostErrs: map[string]error{
				"https://example.com/posts/dup": domain.ErrDuplicatePost,
			},
		}
		fetcher := &fakeFetcher{feeds: map[string]domain.RawFeed{
			feed.URL: {Items: []domain.RawItem{
				{Title: "Inserted",  Link: "https://example.com/posts/ok",  PublishedAt: pub},
				{Title: "Skipped",   Link: "https://example.com/posts/bad", PublishedAt: time.Time{}}, // zero pubDate
				{Title: "Duplicate", Link: "https://example.com/posts/dup", PublishedAt: pub},
			}},
		}}
		svc := NewScrapeService(store, fetcher, &fakeClock{now: now}, newFakeIDGen(postIDs...))

		got, err := svc.Scrape(context.Background())
		if err != nil {
			t.Fatalf("Scrape: %v", err)
		}
		if got.Feed.ID != feed.ID {
			t.Errorf("result.Feed.ID = %v, want %v", got.Feed.ID, feed.ID)
		}
		if got.Found != 3 || got.Inserted != 1 || got.Skipped != 1 || got.Dups != 1 || len(got.Errors) != 0 {
			t.Errorf("counts wrong: Found=%d Inserted=%d Skipped=%d Dups=%d Errors=%d",
				got.Found, got.Inserted, got.Skipped, got.Dups, len(got.Errors))
		}
		if !store.markFetchedAt.Equal(now) {
			t.Errorf("MarkFetched received at=%v, want %v", store.markFetchedAt, now)
		}
		if len(store.createdPosts) != 1 || store.createdPosts[0].Title != "Inserted" {
			t.Errorf("createdPosts = %+v, want exactly the non-duplicate non-skipped item", store.createdPosts)
		}
	})

	t.Run("no feed in DB returns ErrNoFeedToScrape and never touches the fetcher", func(t *testing.T) {
		store := &fakeScrapeStore{nextFeedErr: domain.ErrNoFeedToScrape}
		fetcher := &fakeFetcher{}
		svc := NewScrapeService(store, fetcher, &fakeClock{now: now}, newFakeIDGen())

		_, err := svc.Scrape(context.Background())
		if !errors.Is(err, domain.ErrNoFeedToScrape) {
			t.Fatalf("err = %v, want ErrNoFeedToScrape", err)
		}
		if fetcher.fetchCalls != 0 {
			t.Errorf("fetcher was called %d times, want 0", fetcher.fetchCalls)
		}
	})

	t.Run("MarkFetched failure short-circuits before fetch", func(t *testing.T) {
		store := &fakeScrapeStore{nextFeed: feed, markFetchedErr: errors.New("db down")}
		fetcher := &fakeFetcher{}
		svc := NewScrapeService(store, fetcher, &fakeClock{now: now}, newFakeIDGen())

		got, err := svc.Scrape(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got.Feed.ID != feed.ID {
			t.Errorf("result.Feed should still carry the feed identity on mark failure, got %+v", got.Feed)
		}
		if fetcher.fetchCalls != 0 {
			t.Errorf("fetcher invoked %d times, want 0 (mark-fetched failed)", fetcher.fetchCalls)
		}
	})

	t.Run("fetch failure returns error with the feed identity preserved", func(t *testing.T) {
		store := &fakeScrapeStore{nextFeed: feed}
		fetcher := &fakeFetcher{err: errors.New("network down")}
		svc := NewScrapeService(store, fetcher, &fakeClock{now: now}, newFakeIDGen())

		got, err := svc.Scrape(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got.Feed.ID != feed.ID {
			t.Errorf("result.Feed should carry the feed identity on fetch failure")
		}
		if !store.markFetchedAt.Equal(now) {
			t.Errorf("MarkFetched should still be called before the fetch attempt; markFetchedAt=%v", store.markFetchedAt)
		}
	})

	t.Run("unexpected per-post errors accumulate without aborting the loop", func(t *testing.T) {
		boom := errors.New("disk full")
		store := &fakeScrapeStore{
			nextFeed: feed,
			createPostErrs: map[string]error{
				"https://example.com/posts/boom": boom,
			},
		}
		fetcher := &fakeFetcher{feeds: map[string]domain.RawFeed{
			feed.URL: {Items: []domain.RawItem{
				{Title: "Boom",  Link: "https://example.com/posts/boom", PublishedAt: pub},
				{Title: "After", Link: "https://example.com/posts/after", PublishedAt: pub},
			}},
		}}
		svc := NewScrapeService(store, fetcher, &fakeClock{now: now}, newFakeIDGen(postIDs...))

		got, err := svc.Scrape(context.Background())
		if err != nil {
			t.Fatalf("Scrape returned error %v, want nil (per-item errors accumulate in result.Errors)", err)
		}
		if got.Inserted != 1 || len(got.Errors) != 1 {
			t.Errorf("counts wrong: Inserted=%d Errors=%d, want 1 and 1", got.Inserted, len(got.Errors))
		}
		if !strings.Contains(got.Errors[0].Error(), "Boom") {
			t.Errorf("error message missing item title: %v", got.Errors[0])
		}
		if !errors.Is(got.Errors[0], boom) {
			t.Errorf("error chain lost wrap: %v", got.Errors[0])
		}
	})
}
