package app

import (
	"context"
	"fmt"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
)

// FeedService implements the five feed/follow use-cases: AddFeed,
// ListFeeds, Follow, Unfollow, ListFollows. Per plan decision 6 it does
// NOT consume the Session port — the CLI's RequireLogin middleware
// resolves the current user and passes it explicitly to each method.
type FeedService struct {
	store   ports.FeedStore
	fetcher ports.FeedFetcher
	clock   ports.Clock
	idgen   ports.IDGen
}

func NewFeedService(store ports.FeedStore, fetcher ports.FeedFetcher, clock ports.Clock, idgen ports.IDGen) *FeedService {
	return &FeedService{store: store, fetcher: fetcher, clock: clock, idgen: idgen}
}

// AddFeed validates the URL by fetching it (matching the original handler's
// behavior — this is what makes `addfeed` reject obviously broken URLs),
// then atomically persists the feed and an auto-follow for the owner in a
// single SQL statement (data-modifying CTE). Both rows succeed together or
// neither does; there is no partial-write window.
func (s *FeedService) AddFeed(ctx context.Context, owner domain.User, name, url string) (domain.Feed, error) {
	if _, err := s.fetcher.Fetch(ctx, url); err != nil {
		return domain.Feed{}, fmt.Errorf("failed to fetch feed: %w", err)
	}
	now := s.clock.Now()
	feed, err := s.store.CreateFeedAndFollow(ctx,
		domain.Feed{
			ID:        s.idgen.NewID(),
			CreatedAt: now,
			UpdatedAt: now,
			Name:      name,
			URL:       url,
			UserID:    owner.ID,
		},
		s.idgen.NewID(),
	)
	if err != nil {
		return domain.Feed{}, fmt.Errorf("failed to create feed: %w", err)
	}
	return feed, nil
}

// ListFeeds returns every feed with its owner's name pre-joined.
func (s *FeedService) ListFeeds(ctx context.Context) ([]domain.FeedWithOwner, error) {
	return s.store.ListFeedsWithOwner(ctx)
}

// Follow looks up the feed by URL (returns domain.ErrFeedNotFound if it
// does not exist) and creates a follow row for the owner. The returned
// FeedFollow has FeedName / UserName populated by the underlying joined
// sqlc query, so the CLI can print the confirmation message without a
// second round-trip.
func (s *FeedService) Follow(ctx context.Context, owner domain.User, url string) (domain.FeedFollow, error) {
	feed, err := s.store.GetFeedByURL(ctx, url)
	if err != nil {
		return domain.FeedFollow{}, err
	}
	return s.store.CreateFeedFollow(ctx, domain.FeedFollow{
		ID:        s.idgen.NewID(),
		CreatedAt: s.clock.Now(),
		UserID:    owner.ID,
		FeedID:    feed.ID,
	})
}

// Unfollow removes the (owner, url) follow row and returns the resolved
// feed so the CLI can print "Feed unfollowed: <name> (by <user>)".
func (s *FeedService) Unfollow(ctx context.Context, owner domain.User, url string) (domain.Feed, error) {
	feed, err := s.store.GetFeedByURL(ctx, url)
	if err != nil {
		return domain.Feed{}, err
	}
	if err := s.store.DeleteFeedFollow(ctx, owner.ID, feed.ID); err != nil {
		return domain.Feed{}, err
	}
	return feed, nil
}

// ListFollows returns every follow row for the owner with feed and user
// names pre-joined.
func (s *FeedService) ListFollows(ctx context.Context, owner domain.User) ([]domain.FeedFollow, error) {
	return s.store.ListFeedFollowsByUserID(ctx, owner.ID)
}
