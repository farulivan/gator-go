package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/farulivan/gator-go/internal/database"
	"github.com/google/uuid"
)

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: %s <time_between_reqs>", cmd.name)
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	fmt.Printf("Collecting feeds every %v\n", timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		if err := scrapeFeeds(s); err != nil {
			log.Printf("scrape error: %v", err)
		}
	}
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("Failed to get next feed to fetch: %w", err)
	}

	err = s.db.MarkFeedAsFetched(context.Background(), feed.ID)
	if err != nil {
		return fmt.Errorf("Failed to mark feed as fetched: %w", err)
	}

	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("Failed to fetch feed %s: %w", feed.Name, err)
	}

	fmt.Printf("Fetching feed %s\n", feed.Name)
	for _, item := range rssFeed.Channel.Item {
		formats := []string{time.RFC1123Z, time.RFC1123, time.RFC822Z}
		var publishedAt time.Time
		for _, f := range formats {
			if t, err := time.Parse(f, item.PubDate); err == nil {
				publishedAt = t
				break
			}
		}
		if publishedAt.IsZero() {
			log.Printf("Failed to parse published date for item %s", item.Title)
			continue
		}

		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			Title:       item.Title,
			Url:         item.Link,
			Description: item.Description,
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				continue // skip duplicate posts
			}
			log.Printf("Failed to create post: %v", err)
			continue
		}
	}

	log.Printf("Feed %s collected, %v posts found", feed.Name, len(rssFeed.Channel.Item))
	return nil
}
