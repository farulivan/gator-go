package domain

import "errors"

// Sentinel errors the use-case layer raises and the CLI / driven adapters
// translate to. Adapters are responsible for mapping driver-specific errors
// (sql.ErrNoRows, *pq.Error code 23505, …) onto these sentinels at the
// edges; the use-case layer only ever sees `domain.Err…`.
var (
	// ErrUserNotFound is returned by UserStore.GetUserByName when no user
	// matches the requested name.
	ErrUserNotFound = errors.New("user not found")

	// ErrUserExists is returned by UserStore.CreateUser when the unique
	// constraint on users.name is violated.
	ErrUserExists = errors.New("user already exists")

	// ErrFeedNotFound is returned by FeedStore.GetFeedByURL when no feed
	// matches the requested URL.
	ErrFeedNotFound = errors.New("feed not found")

	// ErrDuplicatePost is returned by ScrapeStore.CreatePost when the
	// unique constraint on posts.url is violated. The scrape use-case
	// uses this to silently skip already-ingested items.
	ErrDuplicatePost = errors.New("duplicate post")

	// ErrNoFeedToScrape is returned by ScrapeStore.GetNextFeedToFetch when
	// the feeds table is empty (no rows to drive a scrape tick).
	ErrNoFeedToScrape = errors.New("no feed to scrape")

	// ErrNotLoggedIn is returned by use-cases that require an authenticated
	// session when the Session port reports an empty current user.
	ErrNotLoggedIn = errors.New("not logged in")

	// ErrFeedExists is returned by FeedStore.CreateFeed when the unique
	// constraint on feeds.url is violated.
	ErrFeedExists = errors.New("feed already exists")

	// ErrAlreadyFollowing is returned by FeedStore.CreateFeedFollow when
	// the unique constraint on feed_follows (user_id, feed_id) is violated.
	ErrAlreadyFollowing = errors.New("already following feed")
)
