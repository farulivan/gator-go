package httprss

import "time"

// pubDateFormats is the historical fallback list gator has accepted in the
// wild. RSS feeds in practice violate the spec, so we try the most common
// variants in order. Returning a zero time.Time is the signal that none of
// them matched.
var pubDateFormats = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
}

// parsePubDate returns the parsed time, or the zero time.Time if the input
// is empty or matches none of the supported formats. The caller is expected
// to skip zero-time items rather than persist them.
func parsePubDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, f := range pubDateFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
