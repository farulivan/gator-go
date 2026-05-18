package httprss

import (
	"testing"
	"time"
)

func TestParsePubDate(t *testing.T) {
	// RSS feeds in the wild violate the spec — these are the three formats
	// pubDateFormats accepts, plus the two failure modes (empty and garbage)
	// the use-case layer uses to decide whether to skip an item.
	cases := []struct {
		name   string
		input  string
		want   time.Time
		isZero bool
	}{
		{
			name:  "RFC1123Z (most common in the wild)",
			input: "Mon, 11 May 2026 09:30:00 +0000",
			want:  time.Date(2026, 5, 11, 9, 30, 0, 0, time.UTC),
		},
		{
			name:  "RFC1123 (named timezone)",
			input: "Mon, 11 May 2026 09:30:00 UTC",
			want:  time.Date(2026, 5, 11, 9, 30, 0, 0, time.UTC),
		},
		{
			name:  "RFC822Z (two-digit year, numeric tz)",
			input: "11 May 26 09:30 +0000",
			want:  time.Date(2026, 5, 11, 9, 30, 0, 0, time.UTC),
		},
		{
			name:   "empty string returns zero time",
			input:  "",
			isZero: true,
		},
		{
			name:   "unparseable string returns zero time",
			input:  "yesterday afternoon",
			isZero: true,
		},
		{
			name:   "ISO 8601 is not in the fallback list, returns zero",
			input:  "2026-05-11T09:30:00Z",
			isZero: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePubDate(tc.input)
			if tc.isZero {
				if !got.IsZero() {
					t.Errorf("parsePubDate(%q) = %v, want zero time", tc.input, got)
				}
				return
			}
			if !got.Equal(tc.want) {
				t.Errorf("parsePubDate(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
