// Package app holds the use-case services that orchestrate domain types
// against the ports. Splitting services out of `internal/domain` (the plan
// originally placed them there) avoids a circular import with
// `internal/ports`, which already depends on the domain types.
package app

import (
	"context"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
)

// UserService implements the four user-management use-cases: Register,
// Login, ListUsers, Reset. Per plan decision 6, it is the only service
// that consumes the Session port — Feed/Browse/Scrape services receive
// the resolved domain.User from the CLI's RequireLogin middleware.
type UserService struct {
	store   ports.UserStore
	clock   ports.Clock
	idgen   ports.IDGen
	session ports.Session
}

func NewUserService(store ports.UserStore, clock ports.Clock, idgen ports.IDGen, session ports.Session) *UserService {
	return &UserService{store: store, clock: clock, idgen: idgen, session: session}
}

// Register creates a new user, then sets them as the current session user.
// Returns domain.ErrUserExists when the name is taken (translated by the
// store adapter from the underlying unique-constraint violation).
func (s *UserService) Register(ctx context.Context, name string) (domain.User, error) {
	user, err := s.store.CreateUser(ctx, domain.User{
		ID:        s.idgen.NewID(),
		CreatedAt: s.clock.Now(),
		Name:      name,
	})
	if err != nil {
		return domain.User{}, err
	}
	if err := s.session.SetCurrentUser(user.Name); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// Login resolves a user by name and sets them as the current session user.
// Returns domain.ErrUserNotFound when no such user exists.
func (s *UserService) Login(ctx context.Context, name string) (domain.User, error) {
	user, err := s.store.GetUserByName(ctx, name)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.session.SetCurrentUser(user.Name); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// ListUsers returns every registered user.
func (s *UserService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.store.ListUsers(ctx)
}

// Reset wipes all users (cascading to feeds, follows, and posts).
func (s *UserService) Reset(ctx context.Context) error {
	return s.store.DeleteAllUsers(ctx)
}

// CurrentUserName surfaces the session's current-user marker for use in
// presentation logic (`users` lists with a `(current)` annotation).
func (s *UserService) CurrentUserName() string {
	return s.session.CurrentUserName()
}
