# Plan: Hexagonal Restructure of `gator-go`

> Persistent copy of the approved refactor plan. Source: `~/.claude/plans/how-about-if-we-sequential-rabin.md`. Resume work from here in future sessions.

## Context

`gator-go` is a small Go CLI RSS aggregator (~11 commands) backed by PostgreSQL via sqlc. The current architecture is flat: handlers (`handler_*.go`) inline argv parsing, sqlc calls, and `fmt.Printf` output. `scrapeFeeds` in `handler_agg.go` mixes HTTP + XML parsing + DB inserts + dup-key string matching. No seams exist for HTTP, time, UUID generation, or even a narrower DB surface — so no part of the codebase is testable without a live Postgres + real network. Time/UUID generation is non-deterministic.

The user has chosen a **full hexagonal (ports & adapters)** restructure with **narrow per-use-case ports**, accepting higher ceremony to gain real test seams and a clean separation between domain logic and infrastructure (CLI / sqlc / HTTP / config file).

Intended outcome: domain logic in `internal/domain` is pure (no `database/sql`, `net/http`, `encoding/xml`, or `internal/database` imports), driven through narrow port interfaces, with adapters under `internal/adapters/*` doing all translation. Tests can fake any port and exercise use-cases in isolation.

## Workflow

### Branch

Create and switch to a feature branch before any code change. Do not work on `main`.

```bash
git checkout main
git pull --ff-only
git checkout -b refactor/hexagonal-architecture
```

Branch name follows the convention used elsewhere on the repo (Conventional Commits-style prefix + kebab-case slug). Alternatives if you prefer something different: `refactor/ports-and-adapters`, `refactor/hexagonal-restructure`. Pick before starting.

### Execution rules

1. Implementation happens **in commit groups** (see the "Commit Groups" table below). Each group bundles tightly-related migration steps into one commit so history reads cleanly.
2. **After finishing each commit group, the agent stops and waits.** The agent does not run `git commit` automatically. It will:
   - Summarise what changed in that group.
   - Suggest a commit message (Conventional Commits format, matching the existing repo style `refactor: …`).
   - Hand control back so the user can review the diff, edit files if needed, and create the commit themselves.
3. The user signals "next group" when ready to continue.
4. After all groups are committed, the agent suggests a **PR title and description** for opening the pull request from `refactor/hexagonal-architecture` → `main`.
5. End-to-end smoke verification (see the Verification section) happens **within each commit group, before the agent pauses**. A group is not "done" until it builds and the relevant commands still work.

### Commit Groups

| # | Migration steps covered | What lands in this commit | Suggested commit message |
|---|---|---|---|
| 1 | Steps 1 + 2 | Package skeleton dirs + extract `FeedFetcher` port and `httprss` adapter. Delete `rss_feed.go`. Existing handlers now call the new adapter via a field on `state`. | `refactor: extract RSS fetcher into httprss adapter behind a port` |
| 2 | Steps 3 + 4 + 5 | Domain types, sentinel errors, all four narrow store ports, all four sqlc adapters with `mapping.go`, plus `Clock`/`IDGen`/`Session` ports and their adapters. Nothing wired yet; build-only. | `refactor: add domain types, narrow ports, and driven adapters (sqlc/clock/id/session)` |
| 3 | Step 6 | `UserService` + CLI router with user handlers wired. Root `main.go` swaps `cmds.register("login", ...)` etc. for the new router. `handler_user.go` deleted. Smoke-test all four user commands. | `refactor: migrate user commands (login/register/users/reset) to UserService via cli.Router` |
| 4 | Step 7 | `FeedService` + feed/follow/following CLI handlers. `handler_feed.go`, `handler_follow.go`, `handler_following.go` deleted. Smoke-test the five feed/follow commands. | `refactor: migrate feed and follow commands to FeedService` |
| 5 | Step 8 | `BrowseService` + `ScrapeService` + CLI handlers (including the agg ticker loop in `handlers_agg.go`). Edit `sql/queries/feeds.sql` so `MarkFeedAsFetched` takes `$2 = at`; run `sqlc generate`. `handler_browse.go`, `handler_agg.go` deleted. Smoke-test `agg`, `browse`. | `refactor: migrate browse and scrape to services; parameterise MarkFeedAsFetched` |
| 6 | Step 9 | Move composition root to `cmd/gator/main.go`. Delete root `main.go`, `commands.go`, `middleware.go` in the same commit. Wire signal-cancelled `ctx` into `router.Run`. Update CLAUDE.md + README build path. | `refactor: move composition root to cmd/gator and complete hexagonal layout` |
| 7 | Step 10 | Domain unit tests in `internal/domain/*_test.go` with in-file fakes (Scrape, User, Feed, Browse services). `httprss` adapter test with `httptest.Server` + canned XML. `pubdate_test.go` table test. | `test: add domain unit tests and httprss adapter test` |
| (verify) | Step 11 | Audit greps (see Verification). No commit. | n/a |

Total: 7 commits on the feature branch.

## Target Layout

```
cmd/gator/main.go                            // composition root (only place that imports both driver + adapters)
internal/
  domain/
    errors.go                                // sentinels: ErrUserNotFound, ErrUserExists, ErrFeedNotFound,
                                             //           ErrDuplicatePost, ErrNoFeedToScrape, ErrNotLoggedIn
    types.go                                 // User, Feed, Post, FeedFollow, FeedWithOwner, PostWithFeed,
                                             //           RawFeed, RawItem  (no xml tags, no sql.NullTime)
    user.go                                  // UserService: Register, Login, ListUsers, Reset
    feed.go                                  // FeedService: AddFeed, ListFeeds, Follow, Unfollow, ListFollows
    browse.go                                // BrowseService: Browse
    scrape.go                                // ScrapeService.Scrape(ctx) (ScrapeResult, error); postIngestor (private)
  ports/
    user_store.go      feed_store.go         // one narrow interface per use-case need
    scrape_store.go    browse_store.go       //   (ScrapeStore = exactly 3 methods)
    fetcher.go         clock.go
    idgen.go           session.go
  adapters/
    sqlc/                                    // wraps *database.Queries; one file per aggregate + mapping.go
      mapping.go user_store.go feed_store.go scrape_store.go browse_store.go
    httprss/                                 // wraps http.Client + xml decode + html.UnescapeString
      fetcher.go pubdate.go                  // pubdate.go owns the RFC1123Z/RFC1123/RFC822Z fallback
    configfile/session.go                    // thin shim around existing internal/config (Read stays in main)
    sysclock/clock.go                        // time.Now().UTC()
    uuidgen/idgen.go                         // uuid.New()
    cli/                                     // argv parse + dispatch + Printf
      command.go router.go middleware.go
      handlers_user.go handlers_feed.go handlers_follow.go
      handlers_following.go handlers_browse.go handlers_agg.go
  config/config.go                           // KEEP — reused via configfile/session.go
  database/*.sql.go                          // KEEP — sqlc-generated; do NOT move (sqlc.yaml `out:` unchanged)
sql/queries/*.sql                            // existing; one edit in step 8
sql/schema/*.sql                             // unchanged
```

### Key design decisions

1. **`internal/database` stays put.** Moving it forces a `sqlc.yaml` rewrite + regenerate, with churn that buys nothing.
2. **Ports split per use-case file** (not a single `repos.go`), matching the narrow-port choice.
3. **Domain types are independent of sqlc.** Adapter `mapping.go` translates `sql.NullTime` → `*time.Time`, drops `UpdatedAt` *(see decision 9)*, renames `Url` → `URL`.
4. **PubDate parsing lives in the HTTP adapter**, not the domain. The fallback list (RFC1123Z → RFC1123 → RFC822Z) is a transport quirk. Bad dates yield zero `time.Time`; use-case skips that item, matching current behavior.
5. **The agg ticker loop lives in the CLI adapter**, not the domain. `ScrapeService.Scrape(ctx)` is one tick; `handlers_agg.go` owns the loop, sleep, and per-tick logging.
6. **Logged-in cross-cut: CLI adapter resolves the current user and passes `domain.User` explicitly to use-cases.** The `Session` port is consumed only by `UserService` (to call `SetCurrentUser` on Login/Register success) and `cli.Router` (to read `CurrentUserName` in `requireLogin` middleware). Domain services for Feed/Browse/Scrape never touch `Session` — keeps the model "session is a CLI affordance, not a domain concept."
7. **Sentinel errors live in `internal/domain/errors.go`** (both use-cases and adapters import `domain` for types, so co-locating is clean). Adapters translate driver errors → sentinels (e.g. `pq.Error` code `23505` → `ErrDuplicatePost`).
8. **Output capture via `io.Writer` on the router.** Handlers use `fmt.Fprintf(r.out, ...)` so tests can buffer. No `fmt.Printf` to `os.Stdout` directly.
9. **Keep `UpdatedAt time.Time` in `domain.Feed`** to preserve current CLI parity (`printFeed` and `printFeedWithUser` print "Updated:"). Persistence-flavored but harmless; avoids a user-visible behavior change.
10. **Edit `MarkFeedAsFetched` SQL to take an `at` timestamp parameter** (currently uses server-side `NOW()`). The port signature is `MarkFetched(ctx, feedID, at)` and the adapter would otherwise ignore `at` — defeats clock injection. One SQL line + `sqlc generate`.

## Critical Files

Modify or create:

- `internal/domain/scrape.go` — most subtle use-case (mark-fetched-before-fetch contract, ingest loop, dup handling)
- `internal/adapters/sqlc/mapping.go` — sqlc↔domain translation, `isDuplicateKey(err)` helper
- `internal/adapters/httprss/fetcher.go` — owns RSS xml struct tags, `User-Agent: gator`, 10s timeout, HTML unescape
- `internal/adapters/cli/router.go` — replaces `commands.go` + `middleware.go` + the registration block in `main.go`
- `cmd/gator/main.go` — new composition root; the only file allowed to import `_ "github.com/lib/pq"` + adapters together

Reuse (do not rewrite):

- `internal/config/config.go` — `Config`, `Read()`, `(*Config).SetUser` are kept; `configfile.Session` wraps them.
- `internal/database/*.sql.go` — sqlc-generated; adapters call these methods.
- `sql/schema/*.sql` and `sql/queries/*.sql` — unchanged except `feeds.sql` (`MarkFeedAsFetched` gains `$2 = at`).
- Existing field/error semantics from current handlers: `pq.Error` code `23505` for user dup (already used in `handler_user.go:49`), `strings.Contains("duplicate key…")` for post dup (in `handler_agg.go:76`).

## Migration Sequence

Each step builds and runs end-to-end except where noted. Steps are bundled into commits per the "Commit Groups" table above; after each commit group the agent pauses for review.

1. **Skeleton dirs** — create empty package dirs; no code. *(Commit 1)*
2. **Extract `FeedFetcher`** — add `domain.RawFeed`/`RawItem`, `ports.FeedFetcher`, `adapters/httprss/`. Wire into existing handlers via a `fetcher` field on `state`. Delete `rss_feed.go`. Smoke: `go run . agg 60s`. *(Commit 1)*
3. **Domain types + sentinels + all 4 store ports** — populate `domain/types.go`, `domain/errors.go`, `ports/{user,feed,scrape,browse}_store.go`. No wiring; build only. *(Commit 2)*
4. **All 4 sqlc adapters + `mapping.go`** — implement each. Build only. *(Commit 2)*
5. **`Clock`, `IDGen`, `Session` ports + adapters** (`sysclock`, `uuidgen`, `configfile`). *(Commit 2)*
6. **`UserService` + CLI router with user handlers** — add `internal/domain/user.go` and `internal/adapters/cli/{command,router,middleware,handlers_user}.go`. In the existing root `main.go`, replace `cmds.register("login", handlerLogin)` etc. with the router. Delete `handler_user.go`. Smoke: `register`, `login`, `users`, `reset`. *(Commit 3)*
7. **`FeedService` + feed/follow handlers** — delete `handler_feed.go`, `handler_follow.go`, `handler_following.go`. Smoke: `addfeed`, `feeds`, `follow`, `unfollow`, `following`. *(Commit 4)*
8. **`BrowseService` + `ScrapeService` + handlers** — delete `handler_browse.go`, `handler_agg.go`. Edit `sql/queries/feeds.sql` so `MarkFeedAsFetched` takes `$2` for `last_fetched_at`; `sqlc generate`. Smoke: `agg 60s`, `browse 5`. *(Commit 5)*
9. **Move main to `cmd/gator/`** — create `cmd/gator/main.go`; delete root `main.go`, `commands.go`, `middleware.go` *in the same commit* (only step where `go build ./...` would briefly fail if split). Update README + CLAUDE.md build command to `go build ./cmd/gator/`. *(Commit 6)*
10. **Tests (greenfield)** — co-located unit tests in `internal/domain/*_test.go` with fakes defined in the test file; `httprss` adapter test using `httptest.Server` + canned RSS XML; `pubdate_test.go` table test. *(Commit 7)*
11. **Audit** — `grep -r "database/sql\|net/http\|encoding/xml\|internal/database" internal/domain internal/ports` must return empty. *(No commit — verification only.)*

## Verification

End-to-end after each migration step (steps 2, 6, 7, 8, 9):

```bash
go build ./...                 # or ./cmd/gator/ after step 9
go run ./cmd/gator/ register testuser
go run ./cmd/gator/ login testuser
go run ./cmd/gator/ addfeed example "https://blog.boot.dev/index.xml"
go run ./cmd/gator/ feeds
go run ./cmd/gator/ following
go run ./cmd/gator/ agg 60s    # let it tick once, Ctrl-C
go run ./cmd/gator/ browse 5
go run ./cmd/gator/ unfollow "https://blog.boot.dev/index.xml"
go run ./cmd/gator/ reset
```

After step 10:

```bash
go test ./internal/domain/...           # use-case unit tests with fakes
go test ./internal/adapters/httprss/... # canned-XML test + pubdate table test
go test ./...                            # everything still green
```

Audit (step 11):

```bash
grep -rE 'database/sql|net/http|encoding/xml|internal/database' internal/domain internal/ports
# expected: no output
grep -rE 'github.com/lib/pq' internal/domain internal/ports internal/adapters
# expected: only internal/adapters/sqlc/*.go (and cmd/gator/main.go for the blank import)
```

## Pull Request

After all seven commits land on `refactor/hexagonal-architecture`, open a PR into `main` with this title and description (final wording may be tightened by the agent at PR time once the actual diff is known):

**Title:**

```
refactor: restructure to hexagonal (ports & adapters) architecture
```

**Description template:**

```markdown
## Summary

- Restructure `gator-go` into a hexagonal architecture with narrow per-use-case ports.
- Pure domain in `internal/domain` (no `database/sql`, `net/http`, or sqlc-generated types leak in).
- Driven adapters under `internal/adapters/{sqlc,httprss,configfile,sysclock,uuidgen}`; driving adapter `internal/adapters/cli` owns argv parsing, dispatch, and output formatting.
- Composition root moved to `cmd/gator/main.go`; build path is now `go build ./cmd/gator/`.
- Parameterise `MarkFeedAsFetched` to accept a timestamp so the `Clock` port is meaningful in tests.
- First unit tests in the repo: `internal/domain/*_test.go` cover Scrape/User/Feed/Browse services with in-file fakes; `internal/adapters/httprss/*_test.go` covers RSS parsing + pubDate fallback.

## Why

Previously every handler inlined argv parsing + sqlc calls + `fmt.Printf`, and `scrapeFeeds` mixed HTTP, XML, DB inserts, and duplicate-key string matching. No part of the codebase was testable without a live Postgres + real network, and time/UUID generation was hard-coded. This restructure adds real seams where they pay rent (scrape pipeline, HTTP fetcher, clock, ID generator) without expanding the user-visible surface.

## Test plan

- [ ] `go build ./cmd/gator/` succeeds.
- [ ] `go test ./...` passes (domain unit tests + httprss adapter test).
- [ ] Manual smoke against a live Postgres:
  - [ ] `gator register <name>` then `gator login <name>` then `gator users` shows `(current)` correctly.
  - [ ] `gator addfeed example "https://blog.boot.dev/index.xml"` then `gator feeds` lists it.
  - [ ] `gator following` shows the new follow.
  - [ ] `gator agg 60s` runs at least one tick and inserts posts; second tick skips duplicates silently.
  - [ ] `gator browse 5` shows posts ordered by `published_at`.
  - [ ] `gator unfollow <url>` removes the follow.
  - [ ] `gator reset` wipes users (and cascades to feeds/follows/posts).
- [ ] `grep -rE 'database/sql|net/http|encoding/xml|internal/database' internal/domain internal/ports` returns no results.
- [ ] `grep -rE 'github.com/lib/pq' internal/domain internal/ports internal/adapters` returns only `internal/adapters/sqlc/*.go` (and `cmd/gator/main.go` for the blank import).
- [ ] CLI output for `feeds` and `addfeed` still includes the `Updated:` line (parity check).

## Notes for reviewers

- `internal/database` (sqlc-generated) is intentionally kept in place; `sqlc.yaml` `out:` is unchanged.
- `internal/config` is reused as-is and wrapped by `configfile.Session`. Renaming/absorbing it can be a follow-up if desired.
- The duplicate-post detection now lives behind `domain.ErrDuplicatePost` (translated from `pq.Error` code `23505` by the sqlc adapter), instead of `strings.Contains` in the use-case.
```

## Risks / Notes

- **`UpdatedAt` parity** kept in `domain.Feed` to avoid a CLI output change.
- **`MarkFeedAsFetched` SQL edit** is required for `Clock` to be meaningful — defer the test for the marking timestamp until that edit ships in step 8.
- **Step 9 (main move)** is the only step where `go build ./...` could fail mid-commit. Add `cmd/gator/main.go` first, delete root `main.go` second, in one commit.
- **`internal/config` rename** is deferred. It still lives at `internal/config/` and `configfile.Session` wraps it. Renaming to `internal/adapters/configfile/` and absorbing `config.go` is a follow-up cleanup; the on-disk file format `~/.gatorconfig.json` is unchanged either way.
- **Pre-built `gator-go` binary** at repo root is already gitignored (`.gitignore:2`). After step 9, rebuild path is `go build -o gator-go ./cmd/gator/` or `go install ./cmd/gator/`.
- **`agg` loop Ctrl-C**: while restructuring, wire a signal-cancelled `ctx` from `cmd/gator/main.go` into `router.Run` so the ticker loop exits cleanly. Minor win; do in step 9.
