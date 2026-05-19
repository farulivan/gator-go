package sqlc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/farulivan/gator-go/internal/database"
	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
	"github.com/google/uuid"
)

// FeedStore adapts *database.Queries to ports.FeedStore.
type FeedStore struct {
	q *database.Queries
}

func NewFeedStore(q *database.Queries) *FeedStore {
	return &FeedStore{q: q}
}

var _ ports.FeedStore = (*FeedStore)(nil)

func (s *FeedStore) CreateFeedAndFollow(ctx context.Context, f domain.Feed, followID uuid.UUID) (domain.Feed, error) {
	created, err := s.q.CreateFeedAndFollow(ctx, database.CreateFeedAndFollowParams{
		FeedID:    f.ID,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
		Name:      f.Name,
		Url:       f.URL,
		UserID:    f.UserID,
		FollowID:  followID,
	})
	if err != nil {
		switch duplicateKeyConstraint(err) {
		case "feeds_url_key":
			return domain.Feed{}, domain.ErrFeedExists
		case "feed_follows_user_id_feed_id_key":
			return domain.Feed{}, domain.ErrAlreadyFollowing
		}
		return domain.Feed{}, err
	}
	return feedFromCreateRow(created), nil
}

func (s *FeedStore) GetFeedByURL(ctx context.Context, url string) (domain.Feed, error) {
	f, err := s.q.GetFeedByURL(ctx, url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Feed{}, domain.ErrFeedNotFound
		}
		return domain.Feed{}, err
	}
	return feedToDomain(f), nil
}

func (s *FeedStore) ListFeedsWithOwner(ctx context.Context) ([]domain.FeedWithOwner, error) {
	rows, err := s.q.GetFeedsWithUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.FeedWithOwner, len(rows))
	for i, r := range rows {
		out[i] = feedWithOwnerToDomain(r)
	}
	return out, nil
}

func (s *FeedStore) CreateFeedFollow(ctx context.Context, ff domain.FeedFollow) (domain.FeedFollow, error) {
	created, err := s.q.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
		ID:        ff.ID,
		CreatedAt: ff.CreatedAt,
		UpdatedAt: ff.CreatedAt,
		UserID:    ff.UserID,
		FeedID:    ff.FeedID,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return domain.FeedFollow{}, domain.ErrAlreadyFollowing
		}
		return domain.FeedFollow{}, err
	}
	return feedFollowFromCreate(created), nil
}

func (s *FeedStore) DeleteFeedFollow(ctx context.Context, userID, feedID uuid.UUID) error {
	return s.q.DeleteFeedFollow(ctx, database.DeleteFeedFollowParams{
		UserID: userID,
		FeedID: feedID,
	})
}

func (s *FeedStore) ListFeedFollowsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.FeedFollow, error) {
	rows, err := s.q.GetFeedFollowsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.FeedFollow, len(rows))
	for i, r := range rows {
		out[i] = feedFollowFromList(r)
	}
	return out, nil
}
