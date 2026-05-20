package ports

import (
	"context"

	"github.com/farulivan/gator-go/internal/domain"
)

// FeedFetcher is the driven port that the use-case layer relies on to pull a
// feed from the outside world. The httprss adapter is the production
// implementation; tests inject a fake.
type FeedFetcher interface {
	Fetch(ctx context.Context, url string) (domain.RawFeed, error)
}
