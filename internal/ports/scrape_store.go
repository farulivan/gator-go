package ports

import (
	"context"
	"time"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/google/uuid"
)

// ScrapeStore is the persistence port for ScrapeService.Scrape — exactly
// three methods, matching the three things one scrape tick has to do:
// pick the next feed, mark it fetched, and ingest its posts.
type ScrapeStore interface {
	// GetNextFeedToFetch returns the feed with the oldest LastFetchedAt
	// (NULLS FIRST), or domain.ErrNoFeedToScrape if there are no feeds.
	GetNextFeedToFetch(ctx context.Context) (domain.Feed, error)

	// MarkFetched records `at` as the feed's last_fetched_at. The
	// underlying SQL is parameterised in commit group 5; until then the
	// adapter ignores `at` and lets the DB use NOW().
	MarkFetched(ctx context.Context, feedID uuid.UUID, at time.Time) error

	// CreatePost persists a single post. Returns domain.ErrDuplicatePost
	// when the unique constraint on posts.url fires, so the use-case can
	// silently skip already-ingested items.
	CreatePost(ctx context.Context, p domain.Post) (domain.Post, error)
}
