package cli

import (
	"context"
	"fmt"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
)

// LoggedInHandler is the signature handlers wrapped by RequireLogin
// receive: the same (ctx, args) plus the resolved domain.User. Per plan
// decision 6 this is how the Feed/Browse/Scrape services consume the
// current user — they never touch the Session port directly.
type LoggedInHandler func(ctx context.Context, args []string, user domain.User) error

// RequireLogin returns a Handler that resolves the current session user
// before invoking the wrapped LoggedInHandler. Two failure modes:
//
//   - empty CurrentUserName → domain.ErrNotLoggedIn
//   - lookup failure (incl. ErrUserNotFound) → wrapped error
//
// The wrapped handler is not invoked in either case.
func RequireLogin(session ports.Session, users ports.UserStore, h LoggedInHandler) Handler {
	return func(ctx context.Context, args []string) error {
		name := session.CurrentUserName()
		if name == "" {
			return domain.ErrNotLoggedIn
		}
		user, err := users.GetUserByName(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		return h(ctx, args, user)
	}
}
