package domain

import (
	"time"

	"github.com/google/uuid"
)

// RawFeed is a feed as fetched from a remote source, after transport-level
// concerns (HTTP, XML, HTML unescape, pubDate parsing) have been resolved.
// The use-case layer consumes RawFeed without caring about the wire format.
type RawFeed struct {
	Title       string
	Link        string
	Description string
	Items       []RawItem
}

// RawItem is a single entry inside a RawFeed. PublishedAt is the zero
// time.Time when the source's pubDate could not be parsed; use-cases are
// expected to skip such items rather than store them with a fake timestamp.
type RawItem struct {
	Title       string
	Link        string
	Description string
	PublishedAt time.Time
}

// User is the domain projection of the persisted users row. UpdatedAt is
// dropped at the boundary (see plan decision 3) because it is never read by
// the use-cases or the CLI; CreatedAt is kept for the (current) handler
// output parity.
type User struct {
	ID        uuid.UUID
	CreatedAt time.Time
	Name      string
}

// Feed is the domain projection of a feeds row. UpdatedAt is intentionally
// kept (plan decision 9) so the existing `feeds` / `addfeed` CLI output
// continues to render an `Updated:` line. LastFetchedAt is `*time.Time` so
// the never-fetched case is unambiguous (sql.NullTime stays inside the
// adapter).
type Feed struct {
	ID            uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Name          string
	URL           string
	UserID        uuid.UUID
	LastFetchedAt *time.Time
}

// Post is the domain projection of a posts row.
type Post struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	Title       string
	URL         string
	Description string
	PublishedAt time.Time
	FeedID      uuid.UUID
}

// FeedFollow is the domain projection of a feed_follows row joined with the
// feed and user names. The Create and List sqlc queries already return the
// joined names, so carrying them on the domain type avoids a second
// round-trip.
type FeedFollow struct {
	ID        uuid.UUID
	CreatedAt time.Time
	UserID    uuid.UUID
	FeedID    uuid.UUID
	FeedName  string
	UserName  string
}

// FeedWithOwner is a Feed enriched with the owning user's display name, as
// returned by `gator feeds`.
type FeedWithOwner struct {
	Feed
	OwnerName string
}

// PostWithFeed is a Post enriched with the originating feed's display name,
// as returned by `gator browse`.
type PostWithFeed struct {
	Post
	FeedName string
}
