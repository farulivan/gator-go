package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/farulivan/gator-go/internal/app"
	"github.com/farulivan/gator-go/internal/domain"
)

// AggHandlers wires `gator agg <interval>` to a ticker loop driving
// ScrapeService.Scrape. The ticker loop lives here (not in the service)
// per plan decision 5 — Scrape is pure "one tick", agg owns the cadence.
type AggHandlers struct {
	out io.Writer
	svc *app.ScrapeService
}

func NewAggHandlers(out io.Writer, svc *app.ScrapeService) *AggHandlers {
	return &AggHandlers{out: out, svc: svc}
}

// Agg parses the interval, fires one scrape immediately, then ticks. The
// loop honours ctx cancellation between ticks; cmd/gator/main.go wires
// signal.NotifyContext so SIGINT/SIGTERM exit cleanly.
func (h *AggHandlers) Agg(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}
	interval, err := time.ParseDuration(args[0])
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	fmt.Fprintf(h.out, "Collecting feeds every %v\n", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := h.tick(ctx); err != nil {
			log.Printf("scrape error: %v", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// tick runs a single scrape and emits the per-tick log lines. The shape
// mirrors the original handler_agg.scrapeFeeds output:
//
//   - "Fetching feed <name>"      — once per tick that found a feed.
//   - per-error log lines         — for non-duplicate insert failures.
//   - "Feed <name> collected, N posts found" — closing line.
//
// Skipped (unparseable pubDate) items are summarised in one line rather
// than logged per-item; the original logged each one with an empty title
// which was noisy without being informative.
func (h *AggHandlers) tick(ctx context.Context) error {
	result, err := h.svc.Scrape(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrNoFeedToScrape) {
			return nil
		}
		return err
	}

	fmt.Fprintf(h.out, "Fetching feed %s\n", result.Feed.Name)
	if result.Skipped > 0 {
		log.Printf("Skipped %d items with unparseable published date", result.Skipped)
	}
	for _, e := range result.Errors {
		log.Printf("Failed to create post: %v", e)
	}
	log.Printf("Feed %s collected, %v posts found", result.Feed.Name, result.Found)
	return nil
}
