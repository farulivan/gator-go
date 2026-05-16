package sqlc

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/farulivan/gator-go/internal/database"
	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
	"github.com/google/uuid"
)

// ScrapeStore adapts *database.Queries to ports.ScrapeStore — exactly three
// methods, matching the per-tick contract of ScrapeService.Scrape.
type ScrapeStore struct {
	q *database.Queries
}

func NewScrapeStore(q *database.Queries) *ScrapeStore {
	return &ScrapeStore{q: q}
}

var _ ports.ScrapeStore = (*ScrapeStore)(nil)

func (s *ScrapeStore) GetNextFeedToFetch(ctx context.Context) (domain.Feed, error) {
	f, err := s.q.GetNextFeedToFetch(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Feed{}, domain.ErrNoFeedToScrape
		}
		return domain.Feed{}, err
	}
	return feedToDomain(f), nil
}

// MarkFetched persists the supplied `at` as the feed's last_fetched_at
// (and updated_at). last_fetched_at is nullable in the schema, so we wrap
// the time in a Valid sql.NullTime before passing it to the generated
// query. Tests can drive a fake clock through this seam.
func (s *ScrapeStore) MarkFetched(ctx context.Context, feedID uuid.UUID, at time.Time) error {
	return s.q.MarkFeedAsFetched(ctx, database.MarkFeedAsFetchedParams{
		ID: feedID,
		At: sql.NullTime{Time: at, Valid: true},
	})
}

func (s *ScrapeStore) CreatePost(ctx context.Context, p domain.Post) (domain.Post, error) {
	created, err := s.q.CreatePost(ctx, database.CreatePostParams{
		ID:          p.ID,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.CreatedAt,
		Title:       p.Title,
		Url:         p.URL,
		Description: p.Description,
		PublishedAt: p.PublishedAt,
		FeedID:      p.FeedID,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return domain.Post{}, domain.ErrDuplicatePost
		}
		return domain.Post{}, err
	}
	return postToDomain(created), nil
}
