package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
)

// ScrapeService is the write-side use-case driving the `agg` ticker. One
// call to Scrape == one tick. The CLI adapter owns the ticker loop and
// the per-tick log lines; the service is just the pipeline.
type ScrapeService struct {
	store   ports.ScrapeStore
	fetcher ports.FeedFetcher
	clock   ports.Clock
	idgen   ports.IDGen
}

func NewScrapeService(store ports.ScrapeStore, fetcher ports.FeedFetcher, clock ports.Clock, idgen ports.IDGen) *ScrapeService {
	return &ScrapeService{store: store, fetcher: fetcher, clock: clock, idgen: idgen}
}

// ScrapeResult is the per-tick summary the CLI uses for logging. Skipped
// counts items with an unparseable PublishedAt (zero time.Time after the
// httprss adapter's pubDate fallback). Dups counts items rejected as
// duplicate by the unique constraint on posts.url. Errors carries any
// other per-item failures (DB connection drops, etc.) so the caller can
// log them; the loop never bails mid-feed.
type ScrapeResult struct {
	Feed     domain.Feed
	Found    int
	Inserted int
	Skipped  int
	Dups     int
	Errors   []error
}

// Scrape performs one tick:
//
//  1. Pick the oldest-fetched feed (or domain.ErrNoFeedToScrape if empty).
//  2. Mark it fetched with the clock-driven `now`.
//  3. Fetch via the FeedFetcher.
//  4. For each item: skip zero-pubDate, persist, classify the error.
//
// The mark-fetched-before-fetch ordering matches the original
// scrapeFeeds: even if the HTTP fetch fails, last_fetched_at advances so
// the next tick rotates to a different feed.
func (s *ScrapeService) Scrape(ctx context.Context) (ScrapeResult, error) {
	feed, err := s.store.GetNextFeedToFetch(ctx)
	if err != nil {
		return ScrapeResult{}, err
	}

	now := s.clock.Now()
	if err := s.store.MarkFetched(ctx, feed.ID, now); err != nil {
		return ScrapeResult{Feed: feed}, err
	}

	raw, err := s.fetcher.Fetch(ctx, feed.URL)
	if err != nil {
		return ScrapeResult{Feed: feed}, err
	}

	result := ScrapeResult{Feed: feed, Found: len(raw.Items)}
	for _, item := range raw.Items {
		if item.PublishedAt.IsZero() {
			result.Skipped++
			continue
		}
		_, err := s.store.CreatePost(ctx, domain.Post{
			ID:          s.idgen.NewID(),
			CreatedAt:   now,
			Title:       item.Title,
			URL:         item.Link,
			Description: item.Description,
			PublishedAt: item.PublishedAt,
			FeedID:      feed.ID,
		})
		switch {
		case err == nil:
			result.Inserted++
		case errors.Is(err, domain.ErrDuplicatePost):
			result.Dups++
		default:
			result.Errors = append(result.Errors, fmt.Errorf("post %q: %w", item.Title, err))
		}
	}
	return result, nil
}
