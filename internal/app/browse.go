package app

import (
	"context"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
)

// BrowseService is the read-side use-case backing `gator browse`. The
// owner is passed in by the CLI's RequireLogin middleware; per plan
// decision 6 the service does not consume Session itself.
type BrowseService struct {
	store ports.BrowseStore
}

func NewBrowseService(store ports.BrowseStore) *BrowseService {
	return &BrowseService{store: store}
}

// Browse returns the most recent posts from feeds the owner follows,
// capped to `limit`. limit is int32 to match the underlying SQL LIMIT
// without an int conversion at every call site.
func (s *BrowseService) Browse(ctx context.Context, owner domain.User, limit int32) ([]domain.PostWithFeed, error) {
	return s.store.GetPostsForUser(ctx, owner.ID, limit)
}
