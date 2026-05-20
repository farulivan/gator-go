package cli

import (
	"context"
	"fmt"

	"github.com/farulivan/gator-go/internal/domain"
)

// Following wires `gator following` to FeedService.ListFollows. Empty
// result prints "No feeds followed" (matches the original handler).
func (h *FeedHandlers) Following(ctx context.Context, args []string, user domain.User) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: following")
	}
	follows, err := h.svc.ListFollows(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to get feed follows: %w", err)
	}
	if len(follows) == 0 {
		fmt.Fprintln(h.out, "No feeds followed")
		return nil
	}
	fmt.Fprintf(h.out, "Feed follows for user: %s\n", user.Name)
	for _, ff := range follows {
		fmt.Fprintf(h.out, "  - %s\n", ff.FeedName)
	}
	return nil
}
