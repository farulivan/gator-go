package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/farulivan/gator-go/internal/app"
	"github.com/farulivan/gator-go/internal/domain"
)

// BrowseHandlers wires `gator browse [limit]` to BrowseService.Browse.
type BrowseHandlers struct {
	out io.Writer
	svc *app.BrowseService
}

func NewBrowseHandlers(out io.Writer, svc *app.BrowseService) *BrowseHandlers {
	return &BrowseHandlers{out: out, svc: svc}
}

// Browse defaults limit to 2 to match the original handler. Output format
// (header + per-post block + separator line) is preserved verbatim — the
// e2e harness asserts on the "from <feed>" substring.
func (h *BrowseHandlers) Browse(ctx context.Context, args []string, user domain.User) error {
	limit := int32(2)
	switch len(args) {
	case 0:
	case 1:
		n, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid limit: %w", err)
		}
		limit = int32(n)
	default:
		return fmt.Errorf("usage: browse [limit]")
	}

	posts, err := h.svc.Browse(ctx, user, limit)
	if err != nil {
		return fmt.Errorf("failed to get posts for user %s: %w", user.Name, err)
	}

	fmt.Fprintf(h.out, "Found %d posts for user %s\n", len(posts), user.Name)
	for _, p := range posts {
		fmt.Fprintf(h.out, "%s from %s\n", p.PublishedAt.Format("Mon Jan 2"), p.FeedName)
		fmt.Fprintf(h.out, "--- %s ---\n", p.Title)
		fmt.Fprintf(h.out, "    %s\n", p.Description)
		fmt.Fprintf(h.out, "Link: %s\n", p.URL)
		fmt.Fprintln(h.out, "=====================================")
	}
	return nil
}
