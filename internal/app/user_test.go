package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/google/uuid"
)

func TestUserService_Register(t *testing.T) {
	now := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	t.Run("happy path persists user and sets session", func(t *testing.T) {
		store := newFakeUserStore()
		clock := &fakeClock{now: now}
		idgen := newFakeIDGen(id)
		session := &fakeSession{}
		svc := NewUserService(store, clock, idgen, session)

		got, err := svc.Register(context.Background(), "alice")
		if err != nil {
			t.Fatalf("Register: %v", err)
		}

		want := domain.User{ID: id, CreatedAt: now, Name: "alice"}
		if got != want {
			t.Errorf("returned user = %+v, want %+v", got, want)
		}
		if persisted, _ := store.GetUserByName(context.Background(), "alice"); persisted != want {
			t.Errorf("persisted user = %+v, want %+v", persisted, want)
		}
		if session.current != "alice" {
			t.Errorf("session.current = %q, want %q", session.current, "alice")
		}
	})

	t.Run("duplicate user surfaces ErrUserExists and does not touch session", func(t *testing.T) {
		store := newFakeUserStore()
		store.users["alice"] = domain.User{ID: id, CreatedAt: now, Name: "alice"}
		session := &fakeSession{}
		svc := NewUserService(store, &fakeClock{now: now}, newFakeIDGen(uuid.New()), session)

		_, err := svc.Register(context.Background(), "alice")
		if !errors.Is(err, domain.ErrUserExists) {
			t.Fatalf("err = %v, want ErrUserExists", err)
		}
		if len(session.setCalls) != 0 {
			t.Errorf("session.SetCurrentUser was called %d times, want 0", len(session.setCalls))
		}
	})

	t.Run("session write error surfaces and is not swallowed", func(t *testing.T) {
		store := newFakeUserStore()
		boom := errors.New("disk full")
		session := &fakeSession{setErr: boom}
		svc := NewUserService(store, &fakeClock{now: now}, newFakeIDGen(id), session)

		_, err := svc.Register(context.Background(), "alice")
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want disk full", err)
		}
	})
}

func TestUserService_Login(t *testing.T) {
	now := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	existing := domain.User{ID: uuid.New(), CreatedAt: now, Name: "alice"}

	t.Run("happy path sets session and returns user", func(t *testing.T) {
		store := newFakeUserStore()
		store.users["alice"] = existing
		session := &fakeSession{}
		svc := NewUserService(store, &fakeClock{now: now}, newFakeIDGen(), session)

		got, err := svc.Login(context.Background(), "alice")
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		if got != existing {
			t.Errorf("got = %+v, want %+v", got, existing)
		}
		if session.current != "alice" {
			t.Errorf("session.current = %q, want alice", session.current)
		}
	})

	t.Run("missing user surfaces ErrUserNotFound", func(t *testing.T) {
		store := newFakeUserStore()
		session := &fakeSession{}
		svc := NewUserService(store, &fakeClock{now: now}, newFakeIDGen(), session)

		_, err := svc.Login(context.Background(), "ghost")
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("err = %v, want ErrUserNotFound", err)
		}
		if len(session.setCalls) != 0 {
			t.Errorf("session.SetCurrentUser was called %d times, want 0", len(session.setCalls))
		}
	})
}

func TestUserService_ListUsers(t *testing.T) {
	store := newFakeUserStore()
	store.users["alice"] = domain.User{Name: "alice"}
	store.users["bob"] = domain.User{Name: "bob"}
	svc := NewUserService(store, &fakeClock{}, newFakeIDGen(), &fakeSession{})

	got, err := svc.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d users, want 2", len(got))
	}
	if got[0].Name != "alice" || got[1].Name != "bob" {
		t.Errorf("got names = [%s %s], want [alice bob]", got[0].Name, got[1].Name)
	}
}

func TestUserService_Reset(t *testing.T) {
	store := newFakeUserStore()
	store.users["alice"] = domain.User{Name: "alice"}
	session := &fakeSession{current: "alice"}
	svc := NewUserService(store, &fakeClock{}, newFakeIDGen(), session)

	if err := svc.Reset(context.Background()); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if got, _ := store.ListUsers(context.Background()); len(got) != 0 {
		t.Errorf("after Reset, users = %v, want empty", got)
	}
	if session.current != "" {
		t.Errorf("after Reset, session.current = %q, want empty", session.current)
	}
	if len(session.setCalls) != 1 || session.setCalls[0] != "" {
		t.Errorf("expected one SetCurrentUser(\"\") call, got %v", session.setCalls)
	}
}

func TestUserService_Reset_SessionWriteFails(t *testing.T) {
	store := newFakeUserStore()
	store.users["alice"] = domain.User{Name: "alice"}
	boom := errors.New("disk full")
	session := &fakeSession{current: "alice", setErr: boom}
	svc := NewUserService(store, &fakeClock{}, newFakeIDGen(), session)

	err := svc.Reset(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want chain to contain disk full", err)
	}
	if got, _ := store.ListUsers(context.Background()); len(got) != 0 {
		t.Errorf("delete should still have happened; users = %v, want empty", got)
	}
}

func TestUserService_CurrentUserName(t *testing.T) {
	session := &fakeSession{current: "alice"}
	svc := NewUserService(newFakeUserStore(), &fakeClock{}, newFakeIDGen(), session)

	if got := svc.CurrentUserName(); got != "alice" {
		t.Errorf("CurrentUserName = %q, want alice", got)
	}
}
