package httprss

import (
	"context"
	"encoding/xml"
	"html"
	"net/http"
	"time"

	"github.com/farulivan/gator-go/internal/domain"
)

// rssFeed and rssItem mirror the on-the-wire RSS shape and stay unexported;
// nothing outside this adapter should depend on the XML structure.
type rssFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// Fetcher implements ports.FeedFetcher backed by net/http and encoding/xml.
type Fetcher struct {
	client *http.Client
}

// New returns a Fetcher with the same HTTP behavior as the original
// rss_feed.go: 10s timeout, User-Agent: gator.
func New() *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Fetch performs the GET, decodes the RSS XML, runs html.UnescapeString on
// any user-visible text, and parses pubDate into a time.Time (zero on
// failure). It does not log; the caller decides what to do with errors and
// zero-time items.
func (f *Fetcher) Fetch(ctx context.Context, url string) (domain.RawFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.RawFeed{}, err
	}
	req.Header.Set("User-Agent", "gator")

	resp, err := f.client.Do(req)
	if err != nil {
		return domain.RawFeed{}, err
	}
	defer resp.Body.Close()

	var raw rssFeed
	if err := xml.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return domain.RawFeed{}, err
	}

	feed := domain.RawFeed{
		Title:       html.UnescapeString(raw.Channel.Title),
		Link:        raw.Channel.Link,
		Description: html.UnescapeString(raw.Channel.Description),
	}
	for _, item := range raw.Channel.Item {
		feed.Items = append(feed.Items, domain.RawItem{
			Title:       html.UnescapeString(item.Title),
			Link:        item.Link,
			Description: html.UnescapeString(item.Description),
			PublishedAt: parsePubDate(item.PubDate),
		})
	}
	return feed, nil
}
