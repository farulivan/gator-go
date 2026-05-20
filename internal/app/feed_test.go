package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/google/uuid"
)

func TestFeedService_AddFeed(t *testing.T) {
	now := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	owner := domain.User{ID: uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"), Name: "alice"}
	feedID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	followID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	t.Run("happy path validates URL, persists feed, and auto-follows owner", func(t *testing.T) {
		store := newFakeFeedStore()
		store.owners[owner.ID] = owner.Name
		fetcher := &fakeFetcher{feeds: map[string]domain.RawFeed{
			"https://example.com/rss": {Title: "Example"},
		}}
		svc := NewFeedService(store, fetcher, &fakeClock{now: now}, newFakeIDGen(feedID, followID))

		feed, err := svc.AddFeed(context.Background(), owner, "Example", "https://example.com/rss")
		if err != nil {
			t.Fatalf("AddFeed: %v", err)
		}
		want := domain.Feed{
			ID:        feedID,
			CreatedAt: now,
			UpdatedAt: now,
			Name:      "Example",
			URL:       "https://example.com/rss",
			UserID:    owner.ID,
		}
		if feed != want {
			t.Errorf("returned feed = %+v, want %+v", feed, want)
		}
		if fetcher.fetchCalls != 1 {
			t.Errorf("fetcher invoked %d times, want exactly 1 (URL validation)", fetcher.fetchCalls)
		}
		if len(store.follows) != 1 {
			t.Fatalf("expected 1 auto-follow, got %d", len(store.follows))
		}
		ff := store.follows[0]
		if ff.ID != followID || ff.UserID != owner.ID || ff.FeedID != feedID {
			t.Errorf("follow row = %+v, want id=%v user=%v feed=%v", ff, followID, owner.ID, feedID)
		}
	})

	t.Run("fetcher failure aborts before any DB write", func(t *testing.T) {
		store := newFakeFeedStore()
		fetcher := &fakeFetcher{err: errors.New("network down")}
		svc := NewFeedService(store, fetcher, &fakeClock{now: now}, newFakeIDGen())

		_, err := svc.AddFeed(context.Background(), owner, "Example", "https://example.com/rss")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to fetch feed") {
			t.Errorf("err = %q, want prefix 'failed to fetch feed'", err)
		}
		if len(store.feeds) != 0 || len(store.follows) != 0 {
			t.Errorf("store touched on fetcher failure: feeds=%d follows=%d", len(store.feeds), len(store.follows))
		}
	})

	t.Run("CreateFeedAndFollow failure surfaces 'failed to create feed'", func(t *testing.T) {
		store := newFakeFeedStore()
		store.createFeedErr = errors.New("db down")
		fetcher := &fakeFetcher{feeds: map[string]domain.RawFeed{"https://example.com/rss": {}}}
		svc := NewFeedService(store, fetcher, &fakeClock{now: now}, newFakeIDGen(feedID, followID))

		_, err := svc.AddFeed(context.Background(), owner, "Example", "https://example.com/rss")
		if err == nil || !strings.Contains(err.Error(), "failed to create feed") {
			t.Errorf("err = %v, want 'failed to create feed'", err)
		}
	})

	t.Run("duplicate URL surfaces ErrFeedExists", func(t *testing.T) {
		store := newFakeFeedStore()
		store.owners[owner.ID] = owner.Name
		store.feeds["https://example.com/rss"] = domain.Feed{URL: "https://example.com/rss"}
		fetcher := &fakeFetcher{feeds: map[string]domain.RawFeed{"https://example.com/rss": {}}}
		svc := NewFeedService(store, fetcher, &fakeClock{now: now}, newFakeIDGen(feedID, followID))

		_, err := svc.AddFeed(context.Background(), owner, "Example", "https://example.com/rss")
		if !errors.Is(err, domain.ErrFeedExists) {
			t.Fatalf("err = %v, want ErrFeedExists", err)
		}
	})
}

func TestFeedService_ListFeeds(t *testing.T) {
	store := newFakeFeedStore()
	ownerID := uuid.New()
	store.owners[ownerID] = "alice"
	store.feeds["https://a.example.com"] = domain.Feed{Name: "A", URL: "https://a.example.com", UserID: ownerID}
	store.feeds["https://b.example.com"] = domain.Feed{Name: "B", URL: "https://b.example.com", UserID: ownerID}
	svc := NewFeedService(store, &fakeFetcher{}, &fakeClock{}, newFakeIDGen())

	got, err := svc.ListFeeds(context.Background())
	if err != nil {
		t.Fatalf("ListFeeds: %v", err)
	}
	if len(got) != 2 || got[0].Name != "A" || got[1].Name != "B" {
		t.Errorf("got %d feeds = %+v, want 2 sorted A,B", len(got), got)
	}
	if got[0].OwnerName != "alice" {
		t.Errorf("OwnerName not joined in: got %q", got[0].OwnerName)
	}
}

func TestFeedService_Follow(t *testing.T) {
	owner := domain.User{ID: uuid.New(), Name: "alice"}
	feedID := uuid.New()
	followID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	now := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)

	t.Run("happy path returns joined FeedFollow", func(t *testing.T) {
		store := newFakeFeedStore()
		store.owners[owner.ID] = owner.Name
		store.feeds["https://example.com/rss"] = domain.Feed{ID: feedID, Name: "Example", URL: "https://example.com/rss"}
		svc := NewFeedService(store, &fakeFetcher{}, &fakeClock{now: now}, newFakeIDGen(followID))

		ff, err := svc.Follow(context.Background(), owner, "https://example.com/rss")
		if err != nil {
			t.Fatalf("Follow: %v", err)
		}
		if ff.ID != followID || ff.FeedID != feedID || ff.UserID != owner.ID {
			t.Errorf("follow ids wrong: %+v", ff)
		}
		if ff.FeedName != "Example" || ff.UserName != "alice" {
			t.Errorf("joined names missing: FeedName=%q UserName=%q", ff.FeedName, ff.UserName)
		}
	})

	t.Run("unknown URL propagates ErrFeedNotFound", func(t *testing.T) {
		store := newFakeFeedStore()
		svc := NewFeedService(store, &fakeFetcher{}, &fakeClock{}, newFakeIDGen())

		_, err := svc.Follow(context.Background(), owner, "https://nope.example.com")
		if !errors.Is(err, domain.ErrFeedNotFound) {
			t.Fatalf("err = %v, want ErrFeedNotFound", err)
		}
	})

	t.Run("duplicate follow surfaces ErrAlreadyFollowing", func(t *testing.T) {
		store := newFakeFeedStore()
		store.owners[owner.ID] = owner.Name
		store.feeds["https://example.com/rss"] = domain.Feed{ID: feedID, URL: "https://example.com/rss"}
		store.follows = []domain.FeedFollow{{UserID: owner.ID, FeedID: feedID}}
		svc := NewFeedService(store, &fakeFetcher{}, &fakeClock{now: now}, newFakeIDGen(followID))

		_, err := svc.Follow(context.Background(), owner, "https://example.com/rss")
		if !errors.Is(err, domain.ErrAlreadyFollowing) {
			t.Fatalf("err = %v, want ErrAlreadyFollowing", err)
		}
	})
}

func TestFeedService_Unfollow(t *testing.T) {
	owner := domain.User{ID: uuid.New(), Name: "alice"}
	feedID := uuid.New()

	t.Run("happy path removes the follow row and returns the resolved feed", func(t *testing.T) {
		store := newFakeFeedStore()
		store.owners[owner.ID] = owner.Name
		store.feeds["https://example.com/rss"] = domain.Feed{ID: feedID, Name: "Example", URL: "https://example.com/rss"}
		store.follows = []domain.FeedFollow{{UserID: owner.ID, FeedID: feedID, FeedName: "Example", UserName: "alice"}}
		svc := NewFeedService(store, &fakeFetcher{}, &fakeClock{}, newFakeIDGen())

		feed, err := svc.Unfollow(context.Background(), owner, "https://example.com/rss")
		if err != nil {
			t.Fatalf("Unfollow: %v", err)
		}
		if feed.Name != "Example" {
			t.Errorf("returned feed = %+v, want Name=Example", feed)
		}
		if len(store.follows) != 0 {
			t.Errorf("follow row not removed: %+v", store.follows)
		}
	})

	t.Run("unknown URL propagates ErrFeedNotFound", func(t *testing.T) {
		store := newFakeFeedStore()
		svc := NewFeedService(store, &fakeFetcher{}, &fakeClock{}, newFakeIDGen())

		_, err := svc.Unfollow(context.Background(), owner, "https://nope.example.com")
		if !errors.Is(err, domain.ErrFeedNotFound) {
			t.Fatalf("err = %v, want ErrFeedNotFound", err)
		}
	})
}

func TestFeedService_ListFollows(t *testing.T) {
	owner := domain.User{ID: uuid.New(), Name: "alice"}
	other := domain.User{ID: uuid.New(), Name: "bob"}
	store := newFakeFeedStore()
	store.follows = []domain.FeedFollow{
		{UserID: owner.ID, FeedName: "A"},
		{UserID: other.ID, FeedName: "B"}, // belongs to a different user — must be filtered out
		{UserID: owner.ID, FeedName: "C"},
	}
	svc := NewFeedService(store, &fakeFetcher{}, &fakeClock{}, newFakeIDGen())

	got, err := svc.ListFollows(context.Background(), owner)
	if err != nil {
		t.Fatalf("ListFollows: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d follows for alice, want 2", len(got))
	}
	if got[0].FeedName != "A" || got[1].FeedName != "C" {
		t.Errorf("filter wrong: %+v", got)
	}
}
