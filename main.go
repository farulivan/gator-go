package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/farulivan/gator-go/internal/adapters/cli"
	"github.com/farulivan/gator-go/internal/adapters/configfile"
	"github.com/farulivan/gator-go/internal/adapters/httprss"
	sqlcadapter "github.com/farulivan/gator-go/internal/adapters/sqlc"
	"github.com/farulivan/gator-go/internal/adapters/sysclock"
	"github.com/farulivan/gator-go/internal/adapters/uuidgen"
	"github.com/farulivan/gator-go/internal/app"
	"github.com/farulivan/gator-go/internal/config"
	"github.com/farulivan/gator-go/internal/database"
	"github.com/farulivan/gator-go/internal/ports"
	_ "github.com/lib/pq"
)

type state struct {
	db      *database.Queries
	cfg     *config.Config
	fetcher ports.FeedFetcher
}

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

	s := &state{
		db:      dbQueries,
		cfg:     &cfg,
		fetcher: httprss.New(),
	}

	// New router for migrated commands. Group 3 covers user verbs;
	// subsequent groups will move feed / follow / agg / browse onto the
	// same router and retire the legacy `cmds` registry below.
	userStore := sqlcadapter.NewUserStore(dbQueries)
	clock := sysclock.New()
	idgen := uuidgen.New()
	session := configfile.New(&cfg)
	userSvc := app.NewUserService(userStore, clock, idgen, session)

	router := cli.NewRouter(os.Stdout)
	userHandlers := cli.NewUserHandlers(router.Out(), userSvc)
	router.Register("login", userHandlers.Login)
	router.Register("register", userHandlers.Register)
	router.Register("users", userHandlers.Users)
	router.Register("reset", userHandlers.Reset)

	// Legacy dispatcher: holds verbs not yet migrated. Empties out across
	// commit groups 4 and 5; deleted in group 6.
	cmds := commands{registeredCommands: make(map[string]func(*state, command) error)}
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	cmds.register("feeds", handlerGetFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("browse", middlewareLoggedIn(handlerBrowse))

	if len(os.Args) < 2 {
		log.Fatal("usage: gator <command> [args...]")
	}

	name := os.Args[1]
	ctx := context.Background()

	if router.Has(name) {
		if err := router.Run(ctx, os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	}

	cmd := command{name: name, args: os.Args[2:]}
	if err := cmds.run(s, cmd); err != nil {
		log.Fatal(err)
	}
}
