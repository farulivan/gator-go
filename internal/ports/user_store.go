package ports

import (
	"context"

	"github.com/farulivan/gator-go/internal/domain"
)

// UserStore is the persistence port the UserService relies on. It is narrow
// on purpose: only the four operations Login/Register/ListUsers/Reset need
// from the users table belong here.
type UserStore interface {
	// GetUserByName returns domain.ErrUserNotFound when no user matches.
	GetUserByName(ctx context.Context, name string) (domain.User, error)

	// CreateUser persists a new user. The use-case layer is responsible
	// for filling in the ID and CreatedAt before calling. Adapters return
	// domain.ErrUserExists on a unique-constraint violation.
	CreateUser(ctx context.Context, u domain.User) (domain.User, error)

	// ListUsers returns every user, in no particular order.
	ListUsers(ctx context.Context) ([]domain.User, error)

	// DeleteAllUsers cascades to feeds, feed_follows, and posts via the
	// schema's ON DELETE CASCADE foreign keys.
	DeleteAllUsers(ctx context.Context) error
}
