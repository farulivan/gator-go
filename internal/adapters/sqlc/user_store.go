package sqlc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/farulivan/gator-go/internal/database"
	"github.com/farulivan/gator-go/internal/domain"
	"github.com/farulivan/gator-go/internal/ports"
)

// UserStore adapts *database.Queries to ports.UserStore.
type UserStore struct {
	q *database.Queries
}

// NewUserStore wires a UserStore over the sqlc Queries handle. The same
// *database.Queries can be shared across all four stores in this package.
func NewUserStore(q *database.Queries) *UserStore {
	return &UserStore{q: q}
}

// Compile-time interface check; surfaces signature drift at build time.
var _ ports.UserStore = (*UserStore)(nil)

func (s *UserStore) GetUserByName(ctx context.Context, name string) (domain.User, error) {
	u, err := s.q.GetUser(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}
	return userToDomain(u), nil
}

func (s *UserStore) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	// The schema's users table requires updated_at; the use-case dropped
	// it from the domain so we re-derive it from CreatedAt at the boundary.
	created, err := s.q.CreateUser(ctx, database.CreateUserParams{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.CreatedAt,
		Name:      u.Name,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return domain.User{}, domain.ErrUserExists
		}
		return domain.User{}, err
	}
	return userToDomain(created), nil
}

func (s *UserStore) ListUsers(ctx context.Context) ([]domain.User, error) {
	users, err := s.q.GetUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.User, len(users))
	for i, u := range users {
		out[i] = userToDomain(u)
	}
	return out, nil
}

func (s *UserStore) DeleteAllUsers(ctx context.Context) error {
	return s.q.DeleteAllUsers(ctx)
}
