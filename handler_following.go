package main

import (
	"context"
	"fmt"

	"github.com/farulivan/gator-go/internal/database"
)

func handlerFollowing(s *state, cmd command, user database.User) error {
	feedFollows, err := s.db.GetFeedFollowsByUserID(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get feed follows: %w", err)
	}

	if len(feedFollows) == 0 {
		fmt.Println("No feeds followed")
		return nil
	}

	fmt.Printf("Feed follows for user: %s\n", user.Name)
	for _, feedFollow := range feedFollows {
		fmt.Printf("  - %s\n", feedFollow.FeedName)
	}

	return nil
}
