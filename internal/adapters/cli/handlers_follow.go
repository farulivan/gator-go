package cli

import (
	"context"
	"fmt"

	"github.com/farulivan/gator-go/internal/domain"
)

// Follow wires `gator follow <url>` to FeedService.Follow. The returned
// FeedFollow already has FeedName/UserName populated by the joined sqlc
// query, so the confirmation message needs no additional lookup.
func (h *FeedHandlers) Follow(ctx context.Context, args []string, user domain.User) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: follow <url>")
	}
	ff, err := h.svc.Follow(ctx, user, args[0])
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %w", err)
	}
	fmt.Fprintf(h.out, "Feed followed: %s (by %s)\n", ff.FeedName, ff.UserName)
	return nil
}

// Unfollow wires `gator unfollow <url>` to FeedService.Unfollow. The
// returned domain.Feed gives us the feed name; the user name comes from
// the resolved current user the RequireLogin middleware passed in.
func (h *FeedHandlers) Unfollow(ctx context.Context, args []string, user domain.User) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: unfollow <url>")
	}
	feed, err := h.svc.Unfollow(ctx, user, args[0])
	if err != nil {
		return fmt.Errorf("failed to delete feed follow: %w", err)
	}
	fmt.Fprintf(h.out, "Feed unfollowed: %s (by %s)\n", feed.Name, user.Name)
	return nil
}
