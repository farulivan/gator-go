package app

import (
	"context"
	"errors"
	"testing"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/google/uuid"
)

func TestBrowseService_Browse(t *testing.T) {
	owner := domain.User{ID: uuid.New(), Name: "alice"}

	t.Run("happy path forwards owner ID and limit and returns posts", func(t *testing.T) {
		want := []domain.PostWithFeed{
			{Post: domain.Post{Title: "A"}, FeedName: "Example"},
			{Post: domain.Post{Title: "B"}, FeedName: "Example"},
		}
		store := &fakeBrowseStore{posts: want}
		svc := NewBrowseService(store)

		got, err := svc.Browse(context.Background(), owner, 5)
		if err != nil {
			t.Fatalf("Browse: %v", err)
		}
		if len(got) != 2 || got[0].Title != "A" || got[1].Title != "B" {
			t.Errorf("got = %+v, want %+v", got, want)
		}
		if store.lastUserID != owner.ID {
			t.Errorf("forwarded userID = %v, want %v", store.lastUserID, owner.ID)
		}
		if store.lastLimit != 5 {
			t.Errorf("forwarded limit = %d, want 5", store.lastLimit)
		}
	})

	t.Run("store error propagates", func(t *testing.T) {
		boom := errors.New("db down")
		store := &fakeBrowseStore{getPostsErr: boom}
		svc := NewBrowseService(store)

		_, err := svc.Browse(context.Background(), owner, 5)
		if !errors.Is(err, boom) {
			t.Errorf("err = %v, want db down", err)
		}
	})
}
