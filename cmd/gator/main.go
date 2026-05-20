// Command gator is the composition root: it constructs the driven
// adapters once, injects them into the use-case services in
// internal/app, and registers every CLI verb on the cli.Router. Nothing
// in internal/{domain,ports,app} depends on this package, so swapping
// adapters (a different DB driver, an in-memory clock, a non-CLI driving
// adapter) only changes wiring here.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/farulivan/gator-go/internal/adapters/cli"
	"github.com/farulivan/gator-go/internal/adapters/configfile"
	"github.com/farulivan/gator-go/internal/adapters/httprss"
	sqlcadapter "github.com/farulivan/gator-go/internal/adapters/sqlc"
	"github.com/farulivan/gator-go/internal/adapters/sysclock"
	"github.com/farulivan/gator-go/internal/adapters/uuidgen"
	"github.com/farulivan/gator-go/internal/app"
	"github.com/farulivan/gator-go/internal/config"
	"github.com/farulivan/gator-go/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		log.Fatalf("error opening database: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	// Driven adapters
	fetcher := httprss.New()
	userStore := sqlcadapter.NewUserStore(dbQueries)
	feedStore := sqlcadapter.NewFeedStore(dbQueries)
	scrapeStore := sqlcadapter.NewScrapeStore(dbQueries)
	browseStore := sqlcadapter.NewBrowseStore(dbQueries)
	clock := sysclock.New()
	idgen := uuidgen.New()
	session := configfile.New(&cfg)

	// Use-case services
	userSvc := app.NewUserService(userStore, clock, idgen, session)
	feedSvc := app.NewFeedService(feedStore, fetcher, clock, idgen)
	scrapeSvc := app.NewScrapeService(scrapeStore, fetcher, clock, idgen)
	browseSvc := app.NewBrowseService(browseStore)

	// CLI router with every verb wired through the new handlers.
	router := cli.NewRouter(os.Stdout)

	userHandlers := cli.NewUserHandlers(router.Out(), userSvc)
	router.Register("login", userHandlers.Login)
	router.Register("register", userHandlers.Register)
	router.Register("users", userHandlers.Users)
	router.Register("reset", userHandlers.Reset)

	feedHandlers := cli.NewFeedHandlers(router.Out(), feedSvc)
	router.Register("addfeed", cli.RequireLogin(session, userStore, feedHandlers.AddFeed))
	router.Register("feeds", feedHandlers.ListFeeds)
	router.Register("follow", cli.RequireLogin(session, userStore, feedHandlers.Follow))
	router.Register("unfollow", cli.RequireLogin(session, userStore, feedHandlers.Unfollow))
	router.Register("following", cli.RequireLogin(session, userStore, feedHandlers.Following))

	aggHandlers := cli.NewAggHandlers(router.Out(), scrapeSvc)
	router.Register("agg", aggHandlers.Agg)

	browseHandlers := cli.NewBrowseHandlers(router.Out(), browseSvc)
	router.Register("browse", cli.RequireLogin(session, userStore, browseHandlers.Browse))

	if len(os.Args) < 2 {
		log.Fatal("usage: gator <command> [args...]")
	}

	// Long-running verbs (currently just agg) check ctx.Done() between
	// ticks. SIGINT/SIGTERM cancel ctx so the loop exits cleanly instead
	// of being killed mid-tick — important because Scrape marks a feed as
	// fetched before fetching, and we want short-running verbs to be
	// uniformly responsive to Ctrl+C too.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := router.Run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
