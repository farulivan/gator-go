package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/farulivan/gator-go/internal/database"
)

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := int32(2)
	if len(cmd.args) == 1 {
		if specifiedLimit, err := strconv.Atoi(cmd.args[0]); err == nil {
			limit = int32(specifiedLimit)
		} else {
			return fmt.Errorf("invalid limit: %w", err)
		}
	}

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return fmt.Errorf("failed to get posts for user %s: %w", user.Name, err)
	}

	fmt.Printf("Found %d posts for user %s\n", len(posts), user.Name)
	for _, post := range posts {
		fmt.Printf("%s from %s\n", post.PublishedAt.Format("Mon Jan 2"), post.FeedName)
		fmt.Printf("--- %s ---\n", post.Title)
		fmt.Printf("    %v\n", post.Description)
		fmt.Printf("Link: %s\n", post.Url)
		fmt.Println("=====================================")
	}
	return nil
}
