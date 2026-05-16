// Package sqlc adapts the sqlc-generated `internal/database` package to the
// narrow ports defined in `internal/ports`. All translation between
// sql.NullTime / `Url` / row types and the domain projections lives here,
// so the use-case layer never sees driver-flavoured types or errors.
package sqlc

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/farulivan/gator-go/internal/database"
	"github.com/farulivan/gator-go/internal/domain"
	"github.com/lib/pq"
)

// userToDomain projects a sqlc users row into domain.User. UpdatedAt is
// dropped at the boundary (decision 3); CreatedAt is kept for parity with
// the existing `register` output.
func userToDomain(u database.User) domain.User {
	return domain.User{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		Name:      u.Name,
	}
}

// feedToDomain projects a sqlc feeds row, translating sql.NullTime into
// *time.Time. UpdatedAt is intentionally preserved (decision 9) so the CLI
// can keep printing the `Updated:` line.
func feedToDomain(f database.Feed) domain.Feed {
	return domain.Feed{
		ID:            f.ID,
		CreatedAt:     f.CreatedAt,
		UpdatedAt:     f.UpdatedAt,
		Name:          f.Name,
		URL:           f.Url,
		UserID:        f.UserID,
		LastFetchedAt: nullTimeToPtr(f.LastFetchedAt),
	}
}

// feedWithOwnerToDomain projects the joined `GetFeedsWithUsers` row.
func feedWithOwnerToDomain(r database.GetFeedsWithUsersRow) domain.FeedWithOwner {
	return domain.FeedWithOwner{
		Feed: domain.Feed{
			ID:            r.ID,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
			Name:          r.Name,
			URL:           r.Url,
			UserID:        r.UserID,
			LastFetchedAt: nullTimeToPtr(r.LastFetchedAt),
		},
		OwnerName: r.UserName,
	}
}

// postToDomain projects a sqlc posts row.
func postToDomain(p database.Post) domain.Post {
	return domain.Post{
		ID:          p.ID,
		CreatedAt:   p.CreatedAt,
		Title:       p.Title,
		URL:         p.Url,
		Description: p.Description,
		PublishedAt: p.PublishedAt,
		FeedID:      p.FeedID,
	}
}

// postWithFeedToDomain projects the joined `GetPostsForUser` row.
func postWithFeedToDomain(r database.GetPostsForUserRow) domain.PostWithFeed {
	return domain.PostWithFeed{
		Post: domain.Post{
			ID:          r.ID,
			CreatedAt:   r.CreatedAt,
			Title:       r.Title,
			URL:         r.Url,
			Description: r.Description,
			PublishedAt: r.PublishedAt,
			FeedID:      r.FeedID,
		},
		FeedName: r.FeedName,
	}
}

// feedFollowFromCreate projects the joined `CreateFeedFollow` row.
func feedFollowFromCreate(r database.CreateFeedFollowRow) domain.FeedFollow {
	return domain.FeedFollow{
		ID:        r.ID,
		CreatedAt: r.CreatedAt,
		UserID:    r.UserID,
		FeedID:    r.FeedID,
		FeedName:  r.FeedName,
		UserName:  r.UserName,
	}
}

// feedFollowFromList projects one row of the joined `GetFeedFollowsByUserID`
// result set.
func feedFollowFromList(r database.GetFeedFollowsByUserIDRow) domain.FeedFollow {
	return domain.FeedFollow{
		ID:        r.ID,
		CreatedAt: r.CreatedAt,
		UserID:    r.UserID,
		FeedID:    r.FeedID,
		FeedName:  r.FeedName,
		UserName:  r.UserName,
	}
}

// nullTimeToPtr collapses sql.NullTime into *time.Time. nil means "never",
// which mirrors how `last_fetched_at` works in the schema.
func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}

// isDuplicateKey returns true when err comes from violating a UNIQUE
// constraint. It checks the typed pq.Error code (23505) first and falls
// back to the substring match the original `handler_agg.scrapeFeeds` used,
// so wrapped errors that have lost the *pq.Error type still classify
// correctly.
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return true
	}
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}
