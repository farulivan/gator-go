package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/farulivan/gator-go/internal/app"
	"github.com/farulivan/gator-go/internal/domain"
)

// FeedHandlers groups the addfeed/feeds/follow/unfollow/following CLI
// verbs. The Follow / Unfollow methods live in handlers_follow.go and
// the Following method lives in handlers_following.go; they all share
// this struct so they consume one writer and one FeedService reference.
type FeedHandlers struct {
	out io.Writer
	svc *app.FeedService
}

func NewFeedHandlers(out io.Writer, svc *app.FeedService) *FeedHandlers {
	return &FeedHandlers{out: out, svc: svc}
}

// AddFeed wires `gator addfeed <name> <url>` to FeedService.AddFeed. The
// service already wraps each underlying error with the original phrasing
// ("failed to fetch feed", "failed to create feed", "feed created
// successfully but failed to create feed follow") so we just propagate.
func (h *FeedHandlers) AddFeed(ctx context.Context, args []string, user domain.User) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: addfeed <name> <url>")
	}
	feed, err := h.svc.AddFeed(ctx, user, args[0], args[1])
	if err != nil {
		if errors.Is(err, domain.ErrFeedExists) {
			return fmt.Errorf("feed URL %q already exists; use `follow %s` to follow it", args[1], args[1])
		}
		return err
	}
	fmt.Fprintln(h.out, "Feed created successfully:")
	printFeed(h.out, feed)
	fmt.Fprintln(h.out)
	fmt.Fprintln(h.out, "=====================================")
	return nil
}

// ListFeeds wires `gator feeds` to FeedService.ListFeeds. Not login-gated
// (matches the original handler).
func (h *FeedHandlers) ListFeeds(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: feeds")
	}
	feeds, err := h.svc.ListFeeds(ctx)
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}
	fmt.Fprintln(h.out, "Feeds:")
	for _, f := range feeds {
		printFeedWithOwner(h.out, f)
		fmt.Fprintln(h.out)
	}
	return nil
}

// printFeed renders a domain.Feed with the same labels and order the
// previous root-package printFeed used. The `Updated:` line is preserved
// per plan decision 9.
func printFeed(out io.Writer, f domain.Feed) {
	fmt.Fprintf(out, "* ID:            %s\n", f.ID)
	fmt.Fprintf(out, "* Created:       %v\n", f.CreatedAt)
	fmt.Fprintf(out, "* Updated:       %v\n", f.UpdatedAt)
	fmt.Fprintf(out, "* Name:          %s\n", f.Name)
	fmt.Fprintf(out, "* URL:           %s\n", f.URL)
	fmt.Fprintf(out, "* UserID:        %s\n", f.UserID)
	fmt.Fprintf(out, "* Last Fetched:  %s\n", formatLastFetched(f.LastFetchedAt))
}

// printFeedWithOwner is printFeed plus the owner's display name, matching
// the previous printFeedWithUser shape.
func printFeedWithOwner(out io.Writer, f domain.FeedWithOwner) {
	fmt.Fprintf(out, "* ID:            %s\n", f.ID)
	fmt.Fprintf(out, "* Created:       %v\n", f.CreatedAt)
	fmt.Fprintf(out, "* Updated:       %v\n", f.UpdatedAt)
	fmt.Fprintf(out, "* Name:          %s\n", f.Name)
	fmt.Fprintf(out, "* URL:           %s\n", f.URL)
	fmt.Fprintf(out, "* UserID:        %s\n", f.UserID)
	fmt.Fprintf(out, "* User Name:     %s\n", f.OwnerName)
	fmt.Fprintf(out, "* Last Fetched:  %s\n", formatLastFetched(f.LastFetchedAt))
}

// formatLastFetched renders a never-fetched feed as "never" rather than
// the previous sql.NullTime sentinel ("{0001-01-01 00:00:00 +0000 UTC
// false}"). Already-fetched feeds print the same time.Time format as
// before via Go's default %v formatter.
func formatLastFetched(t *time.Time) string {
	if t == nil {
		return "never"
	}
	return fmt.Sprintf("%v", *t)
}
