package ports

import (
	"context"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/google/uuid"
)

// FeedStore is the persistence port for the FeedService (addfeed, feeds,
// follow, unfollow, following). The Create / List feed-follow methods
// return domain.FeedFollow with FeedName/UserName already resolved because
// the underlying sqlc queries do the join.
type FeedStore interface {
	// CreateFeedAndFollow inserts a feed and an auto-follow for its owner
	// atomically (single SQL statement; both rolled back if either unique
	// constraint fires). Returns:
	//   - domain.ErrFeedExists if feeds.url is taken
	//   - domain.ErrAlreadyFollowing if the (user, feed) follow row exists
	//     (structurally impossible for a brand-new feed, translated
	//     defensively for completeness)
	CreateFeedAndFollow(ctx context.Context, f domain.Feed, followID uuid.UUID) (domain.Feed, error)

	// GetFeedByURL returns domain.ErrFeedNotFound when no row matches.
	GetFeedByURL(ctx context.Context, url string) (domain.Feed, error)

	// ListFeedsWithOwner returns every feed joined with its owner's name.
	ListFeedsWithOwner(ctx context.Context) ([]domain.FeedWithOwner, error)

	// CreateFeedFollow persists a follow row and returns the joined view
	// (FeedName / UserName populated).
	CreateFeedFollow(ctx context.Context, ff domain.FeedFollow) (domain.FeedFollow, error)

	// DeleteFeedFollow removes the (userID, feedID) follow row, no-op if
	// the row does not exist.
	DeleteFeedFollow(ctx context.Context, userID, feedID uuid.UUID) error

	// ListFeedFollowsByUserID returns every follow for the user, with the
	// feed and user names already joined in.
	ListFeedFollowsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.FeedFollow, error)
}
