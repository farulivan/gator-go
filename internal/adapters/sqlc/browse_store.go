package sqlc

import (
	"context"

	"github.com/farulivan/gator-go/internal/database"
	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
	"github.com/google/uuid"
)

// BrowseStore adapts *database.Queries to ports.BrowseStore. Single
// method: list the most recent posts from feeds the user follows.
type BrowseStore struct {
	q *database.Queries
}

func NewBrowseStore(q *database.Queries) *BrowseStore {
	return &BrowseStore{q: q}
}

var _ ports.BrowseStore = (*BrowseStore)(nil)

func (s *BrowseStore) GetPostsForUser(ctx context.Context, userID uuid.UUID, limit int32) ([]domain.PostWithFeed, error) {
	rows, err := s.q.GetPostsForUser(ctx, database.GetPostsForUserParams{
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.PostWithFeed, len(rows))
	for i, r := range rows {
		out[i] = postWithFeedToDomain(r)
	}
	return out, nil
}
