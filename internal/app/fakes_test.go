package app

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/farulivan/gator-go/internal/domain"
	"github.com/google/uuid"
)

// Fake adapters used across the service tests. They are deliberately
// dumb — just enough state to assert the use-case behavior — and live in
// a single _test.go file so each service test can compose them without
// re-declaring boilerplate.

// --- Clock ---------------------------------------------------------------

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

// --- IDGen ---------------------------------------------------------------

// fakeIDGen hands out a deterministic sequence of UUIDs. The test calls
// newFakeIDGen with however many ids the use-case will mint; running off
// the end panics rather than silently returning the zero UUID, so a test
// that asks for an unexpected number of ids fails loudly.
type fakeIDGen struct {
	ids []uuid.UUID
	i   int
}

func newFakeIDGen(ids ...uuid.UUID) *fakeIDGen {
	return &fakeIDGen{ids: ids}
}

func (g *fakeIDGen) NewID() uuid.UUID {
	if g.i >= len(g.ids) {
		panic("fakeIDGen: out of pre-seeded ids")
	}
	id := g.ids[g.i]
	g.i++
	return id
}

// --- Session -------------------------------------------------------------

type fakeSession struct {
	current  string
	setCalls []string // every name SetCurrentUser was invoked with
	setErr   error
}

func (s *fakeSession) CurrentUserName() string { return s.current }
func (s *fakeSession) SetCurrentUser(name string) error {
	s.setCalls = append(s.setCalls, name)
	if s.setErr != nil {
		return s.setErr
	}
	s.current = name
	return nil
}

// --- UserStore -----------------------------------------------------------

type fakeUserStore struct {
	users     map[string]domain.User
	createErr error // forced error on CreateUser
	listErr   error
	deleteErr error
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{users: make(map[string]domain.User)}
}

func (s *fakeUserStore) CreateUser(_ context.Context, u domain.User) (domain.User, error) {
	if s.createErr != nil {
		return domain.User{}, s.createErr
	}
	if _, ok := s.users[u.Name]; ok {
		return domain.User{}, domain.ErrUserExists
	}
	s.users[u.Name] = u
	return u, nil
}

func (s *fakeUserStore) GetUserByName(_ context.Context, name string) (domain.User, error) {
	u, ok := s.users[name]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (s *fakeUserStore) ListUsers(_ context.Context) ([]domain.User, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]domain.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	// Deterministic ordering keeps table tests predictable.
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *fakeUserStore) DeleteAllUsers(_ context.Context) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.users = map[string]domain.User{}
	return nil
}

// --- FeedStore -----------------------------------------------------------

type fakeFeedStore struct {
	feeds         map[string]domain.Feed // keyed by URL
	follows       []domain.FeedFollow
	owners        map[uuid.UUID]string // userID -> name, used to populate FeedFollow.UserName on Create
	createFeedErr error
	createFFErr   error
	deleteFFErr   error
	listErr       error
	listFollowErr error
}

func newFakeFeedStore() *fakeFeedStore {
	return &fakeFeedStore{
		feeds:  make(map[string]domain.Feed),
		owners: make(map[uuid.UUID]string),
	}
}

func (s *fakeFeedStore) CreateFeed(_ context.Context, f domain.Feed) (domain.Feed, error) {
	if s.createFeedErr != nil {
		return domain.Feed{}, s.createFeedErr
	}
	if _, exists := s.feeds[f.URL]; exists {
		return domain.Feed{}, domain.ErrFeedExists
	}
	s.feeds[f.URL] = f
	return f, nil
}

func (s *fakeFeedStore) GetFeedByURL(_ context.Context, url string) (domain.Feed, error) {
	f, ok := s.feeds[url]
	if !ok {
		return domain.Feed{}, domain.ErrFeedNotFound
	}
	return f, nil
}

func (s *fakeFeedStore) ListFeedsWithOwner(_ context.Context) ([]domain.FeedWithOwner, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]domain.FeedWithOwner, 0, len(s.feeds))
	for _, f := range s.feeds {
		out = append(out, domain.FeedWithOwner{Feed: f, OwnerName: s.owners[f.UserID]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *fakeFeedStore) CreateFeedFollow(_ context.Context, ff domain.FeedFollow) (domain.FeedFollow, error) {
	if s.createFFErr != nil {
		return domain.FeedFollow{}, s.createFFErr
	}
	for _, existing := range s.follows {
		if existing.UserID == ff.UserID && existing.FeedID == ff.FeedID {
			return domain.FeedFollow{}, domain.ErrAlreadyFollowing
		}
	}
	// Populate the joined fields the way the real sqlc query would.
	for _, f := range s.feeds {
		if f.ID == ff.FeedID {
			ff.FeedName = f.Name
			break
		}
	}
	ff.UserName = s.owners[ff.UserID]
	s.follows = append(s.follows, ff)
	return ff, nil
}

func (s *fakeFeedStore) DeleteFeedFollow(_ context.Context, userID, feedID uuid.UUID) error {
	if s.deleteFFErr != nil {
		return s.deleteFFErr
	}
	out := s.follows[:0]
	for _, ff := range s.follows {
		if ff.UserID == userID && ff.FeedID == feedID {
			continue
		}
		out = append(out, ff)
	}
	s.follows = out
	return nil
}

func (s *fakeFeedStore) ListFeedFollowsByUserID(_ context.Context, userID uuid.UUID) ([]domain.FeedFollow, error) {
	if s.listFollowErr != nil {
		return nil, s.listFollowErr
	}
	var out []domain.FeedFollow
	for _, ff := range s.follows {
		if ff.UserID == userID {
			out = append(out, ff)
		}
	}
	return out, nil
}

// --- ScrapeStore ---------------------------------------------------------

type fakeScrapeStore struct {
	nextFeed       domain.Feed
	nextFeedErr    error
	markFetchedAt  time.Time // last `at` MarkFetched saw
	markFetchedErr error
	createPostErrs map[string]error // keyed by post URL — to inject ErrDuplicatePost or arbitrary errors
	createdPosts   []domain.Post
}

func (s *fakeScrapeStore) GetNextFeedToFetch(_ context.Context) (domain.Feed, error) {
	if s.nextFeedErr != nil {
		return domain.Feed{}, s.nextFeedErr
	}
	return s.nextFeed, nil
}

func (s *fakeScrapeStore) MarkFetched(_ context.Context, _ uuid.UUID, at time.Time) error {
	if s.markFetchedErr != nil {
		return s.markFetchedErr
	}
	s.markFetchedAt = at
	return nil
}

func (s *fakeScrapeStore) CreatePost(_ context.Context, p domain.Post) (domain.Post, error) {
	if err, ok := s.createPostErrs[p.URL]; ok {
		return domain.Post{}, err
	}
	s.createdPosts = append(s.createdPosts, p)
	return p, nil
}

// --- BrowseStore ---------------------------------------------------------

type fakeBrowseStore struct {
	posts       []domain.PostWithFeed
	lastUserID  uuid.UUID
	lastLimit   int32
	getPostsErr error
}

func (s *fakeBrowseStore) GetPostsForUser(_ context.Context, userID uuid.UUID, limit int32) ([]domain.PostWithFeed, error) {
	s.lastUserID = userID
	s.lastLimit = limit
	if s.getPostsErr != nil {
		return nil, s.getPostsErr
	}
	return s.posts, nil
}

// --- FeedFetcher ---------------------------------------------------------

type fakeFetcher struct {
	feeds      map[string]domain.RawFeed
	err        error
	lastURL    string
	fetchCalls int
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) (domain.RawFeed, error) {
	f.lastURL = url
	f.fetchCalls++
	if f.err != nil {
		return domain.RawFeed{}, f.err
	}
	feed, ok := f.feeds[url]
	if !ok {
		return domain.RawFeed{}, errors.New("fakeFetcher: no feed seeded for " + url)
	}
	return feed, nil
}
