package ports

import (
	"context"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/google/uuid"
)

// BrowseStore is the persistence port for BrowseService.Browse. Single
// method: list the most recent posts from feeds the user follows.
type BrowseStore interface {
	GetPostsForUser(ctx context.Context, userID uuid.UUID, limit int32) ([]domain.PostWithFeed, error)
}
