package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/farulivan/gator-go/internal/app"
	"github.com/farulivan/gator-go/internal/domain"
)

// UserHandlers groups the four user-management CLI verbs (login, register,
// users, reset) so they can share a writer and a single UserService
// reference rather than each handler being a one-off closure.
type UserHandlers struct {
	out io.Writer
	svc *app.UserService
}

func NewUserHandlers(out io.Writer, svc *app.UserService) *UserHandlers {
	return &UserHandlers{out: out, svc: svc}
}

// Login wires `gator login <name>` to UserService.Login.
func (h *UserHandlers) Login(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: login <name>")
	}
	user, err := h.svc.Login(ctx, args[0])
	if err != nil {
		return fmt.Errorf("couldn't get user: %w", err)
	}
	fmt.Fprintf(h.out, "User has been set to: %s\n", user.Name)
	return nil
}

// Register wires `gator register <name>` to UserService.Register and
// translates ErrUserExists into the same human-readable phrase the
// previous handler emitted.
func (h *UserHandlers) Register(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: register <name>")
	}
	user, err := h.svc.Register(ctx, args[0])
	if err != nil {
		if errors.Is(err, domain.ErrUserExists) {
			return fmt.Errorf("username '%s' already exists", args[0])
		}
		return fmt.Errorf("couldn't create user: %w", err)
	}
	fmt.Fprintf(h.out, "User created: %v\n", user)
	fmt.Fprintf(h.out, "User has been set to: %s\n", user.Name)
	return nil
}

// Users wires `gator users` to UserService.ListUsers, marking the
// session's current user with `(current)` to match the original output.
func (h *UserHandlers) Users(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: users")
	}
	users, err := h.svc.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}
	current := h.svc.CurrentUserName()
	fmt.Fprintln(h.out, "Users:")
	for _, u := range users {
		if u.Name == current {
			fmt.Fprintf(h.out, "  * %s (current)\n", u.Name)
		} else {
			fmt.Fprintf(h.out, "  * %s\n", u.Name)
		}
	}
	return nil
}

// Reset wires `gator reset` to UserService.Reset. The DELETE cascades to
// feeds/follows/posts via the schema's foreign keys.
func (h *UserHandlers) Reset(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: reset")
	}
	if err := h.svc.Reset(ctx); err != nil {
		return fmt.Errorf("failed to delete all users: %w", err)
	}
	fmt.Fprintln(h.out, "All users have been deleted")
	return nil
}
