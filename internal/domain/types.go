package domain

import "time"

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
