# Plan: Polish & Hardening Follow-ups for Hexagonal Architecture

> Generated from the senior-architect review of branch `chore/post-refactor-followups` (off `refactor/hexagonal-architecture`). Pairs with `docs/hexagonal-restructure-plan.md` and the in-flight plan file at `/Users/farul/.claude/plans/memoized-swinging-gizmo.md`. Targets a follow-up branch after `chore/post-refactor-followups` lands.

## Context

The hexagonal restructure shipped clean: boundaries hold under audit, all unit tests pass, and the three post-refactor commits (B/C1/C2 findings) closed every architectural gap. A fresh whole-project review surfaced a few categories of remaining improvement opportunities — **none are blockers, none change architecture, all are small** — that are worth landing as one focused branch before declaring the refactor effort done.

The review grouped findings into four buckets. Items in this plan are the ones worth fixing now:

- **B-list** — five small code-quality nits. Total diff ≈ 30 lines.
- **C-list** — pre-existing UX/log carry-overs that pre-date the refactor but become easy to fix now that the architecture has the right seams.
- **D-list** — test-coverage gaps. CLI handler tests are the biggest one.
- **E-list** — production-hardening (structured logging, retries, file locking). **Out of scope for this plan**; documented in the "Explicitly deferred" section so future planners don't re-discover them.

This plan picks the items where the fix is small, the value is real, and the architecture already supports the change. Items explicitly deferred to follow-up work are listed at the end with reasoning.

## Workflow

### Branch

Start the branch off the merge-base of whichever branch contains the C1/C2 follow-ups. At the time of writing that is `chore/post-refactor-followups`, but if it has merged to `main` or back into `refactor/hexagonal-architecture`, pick whichever is current.

```bash
git checkout chore/post-refactor-followups   # or main / refactor branch
git pull --ff-only
git checkout -b chore/polish-and-hardening
```

### Execution rules

1. Eight commits, in the order below. Each one builds, vets, and tests cleanly on its own — verify before moving on.
2. After each commit group's edits, run:
   ```bash
   go build ./...
   go vet ./...
   go test ./internal/... -count=1
   ```
   No new test should be allowed to use the existing test cache; pass `-count=1` so renames don't fall through.
3. **Stage specific files** — `git add <path>`. Do not `git add .` or `-A`.
4. Commit messages follow Conventional Commits with the emoji suffix this repo uses: `refactor:` and `chore:` → 🏗️ or 🧹, `fix:` → 🐛, `test:` → 🧪, `feat:` → ✨.
5. Do **not** include a `Co-Authored-By` trailer unless the user explicitly asks — recent commits on this repo don't carry one.
6. Do not push; the user reviews locally and pushes when ready.

### Commit groups overview

| # | Theme | What lands | Suggested subject |
|---|---|---|---|
| 1 | B2 + B3 + B4 polish | Drop dead `createFFErr` field in fakes; pass-through `ErrFeedExists` in `AddFeed` (don't wrap with misleading "failed to create feed"); split `s.idgen.NewID()` into two named locals for readability. | `refactor: polish AddFeed error path and remove dead test field 🏗️` |
| 2 | B5 — substring fallback | Remove the substring-match branch in `isDuplicateKey` (dead in this codebase). Rewrite the comment to document the typed-only contract. | `refactor: drop dead substring fallback in isDuplicateKey 🧹` |
| 3 | C-A — unfollow missing row | New `ErrNotFollowing` sentinel; change `DeleteFeedFollow` SQL annotation to `:execrows`; adapter returns sentinel when 0 rows affected; service propagates; handler translates to friendly message. | `fix: surface ErrNotFollowing when unfollow targets a non-existent row 🐛` |
| 4 | C-B — agg log clarity | Change the per-tick closing log line from "N posts found" (which is misleading when some inserts failed) to a summary of `Found/Inserted/Dups/Skipped/Errors`. | `chore: agg per-tick log summary shows all 5 counters 🧹` |
| 5 | C-D — input validation | Add light validation in `FeedService.AddFeed`, `Follow`, `Unfollow` and `UserService.Register`, `Login`: trim whitespace, reject empty. Use plain `fmt.Errorf` (no sentinel — these are caller-side bugs, not domain conditions). | `feat: validate non-empty name and url inputs in user/feed services ✨` |
| 6 | D2 — mapping unit test | New `internal/adapters/sqlc/mapping_test.go` covering `isDuplicateKey` (after B5 simplifies it) and `duplicateKeyConstraint` against fake `*pq.Error` values. | `test: cover isDuplicateKey and duplicateKeyConstraint helpers 🧪` |
| 7 | D1 — CLI handler tests | New `internal/adapters/cli/handlers_*_test.go` files. Stub services satisfying the methods each handler calls; `bytes.Buffer` writer; assert output and error wording. Covers usage strings, ErrFeedExists/ErrAlreadyFollowing/ErrNotFollowing translation, RequireLogin's two failure modes. | `test: add unit tests for CLI handlers and RequireLogin 🧪` |
| 8 | D3 — e2e harness | Extend `e2e/e2e_test.go` to run `addfeed` twice and assert the second produces the new "feed URL ... already exists" message. Same for `follow` (already-following). | `test: extend e2e harness to cover friendly duplicate-write messages 🧪` |

Total: 8 commits. Estimated diff: ~400 lines (most in the new test files).

---

## Commit 1 — B2 + B3 + B4: polish AddFeed plumbing

### Goal

Three independent small fixes to AddFeed and its fake: drop dead field, fix misleading error wrap, make the two-NewID-call dependency explicit.

### Files

- `internal/app/feed.go` — extract IDs to named locals; conditional wrap
- `internal/app/fakes_test.go` — remove `createFFErr` field
- `internal/app/feed_test.go` — adjust any reference to `createFFErr` (none currently set it; check before committing)

### Changes

**B4: extract feed and follow IDs to locals.** Replace lines 35-46 of `internal/app/feed.go` (`AddFeed`) so the two `NewID()` calls are explicit:

```go
now := s.clock.Now()
feedID := s.idgen.NewID()
followID := s.idgen.NewID()
feed, err := s.store.CreateFeedAndFollow(ctx,
    domain.Feed{
        ID:        feedID,
        CreatedAt: now,
        UpdatedAt: now,
        Name:      name,
        URL:       url,
        UserID:    owner.ID,
    },
    followID,
)
```

This works identically to the inline version (Go evaluates function arguments left-to-right per spec) but spells out the ordering for future readers.

**B3: pass `ErrFeedExists` through without the misleading wrap.** Replace the error block after the call:

```go
if err != nil {
    if errors.Is(err, domain.ErrFeedExists) {
        return domain.Feed{}, err
    }
    return domain.Feed{}, fmt.Errorf("failed to create feed: %w", err)
}
```

Add `"errors"` to the import block of `internal/app/feed.go`. The CLI handler already does `errors.Is(err, domain.ErrFeedExists)` so the bare sentinel propagates correctly.

**B2: drop the dead `createFFErr` field.** In `internal/app/fakes_test.go`, remove the line from the struct declaration around line 128:

```go
createFFErr   error    // ← delete this line
```

And remove the `if s.createFFErr != nil { ... }` guard in `fakeFeedStore.CreateFeedFollow` (lines 181-183). Verify no other test references it (grep `createFFErr` in `internal/app/`). If a future test wants to inject a follow-side error, re-add the field at that point.

### Verification

```bash
go build ./...
go vet ./...
go test ./internal/... -count=1
```

All existing tests must still pass; the change is behavior-preserving.

### Suggested commit message

```
refactor: polish AddFeed error path and remove dead test field 🏗️

Three small cleanups in the AddFeed flow:

- Pass ErrFeedExists through bare instead of wrapping with "failed to
  create feed". The CLI handler already translates the sentinel to a
  user-facing message; the wrap text in the chain was misleading
  (feed wasn't created — it already existed).
- Extract feedID and followID into named locals so the two-NewID-call
  dependency in AddFeed is explicit rather than implicit in argument
  ordering.
- Drop the createFFErr field from fakeFeedStore. No test sets it
  after the C2 commit removed the partial-failure test path.
```

---

## Commit 2 — B5: remove the dead substring fallback

### Goal

`isDuplicateKey` in `internal/adapters/sqlc/mapping.go` falls back to substring-matching `"duplicate key value violates unique constraint"` after the typed `*pq.Error` check fails. In this codebase, sqlc-generated functions return raw `*pq.Error` values — they're never wrapped before classification. The fallback can never trigger. Remove and document the typed-only contract.

### Files

- `internal/adapters/sqlc/mapping.go`

### Changes

Replace the current `isDuplicateKey` (lines 139-153):

```go
// isDuplicateKey returns true when err comes from a typed *pq.Error
// with code 23505 (unique-key violation). Callers in this package
// receive raw pq errors from the sqlc-generated layer, so the typed
// check is sufficient — no substring fallback needed. If future code
// starts wrapping errors before they reach the adapter, this helper
// will need a fallback (or re-think) to stay correct.
func isDuplicateKey(err error) bool {
    if err == nil {
        return false
    }
    var pqErr *pq.Error
    return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
```

Remove the `"strings"` import from the import block if no other function in the file uses it. (Run `go vet` — it will complain about an unused import.)

### Verification

```bash
go build ./...
go vet ./...
go test ./internal/... -count=1
```

The `feed_test.go` and `user_test.go` paths that use `ErrUserExists`, `ErrFeedExists`, etc. still pass — they trigger the typed branch.

### Suggested commit message

```
refactor: drop dead substring fallback in isDuplicateKey 🧹

The substring-match fallback was carried over from the pre-refactor
handler_agg.scrapeFeeds duplicate-key check. In the current adapter
layout, sqlc-generated functions return raw *pq.Error values, so the
typed check at the top of isDuplicateKey is always sufficient. The
fallback is unreachable; remove it and document the typed-only
contract so a future wrapper-introducer knows what to revisit.
```

---

## Commit 3 — C-A: surface `ErrNotFollowing` on missing-row unfollow

### Goal

`gator unfollow https://nope.example.com` currently prints "Feed unfollowed: ..." even when no follow row existed for the user. Translate the no-rows-affected case into a domain sentinel and a friendly CLI message.

### Files

- `sql/queries/feed_follows.sql` — change `DeleteFeedFollow` annotation from `:exec` to `:execrows`
- `internal/database/feed_follows.sql.go` — regenerated by `sqlc generate`
- `internal/domain/errors.go` — new `ErrNotFollowing` sentinel
- `internal/ports/feed_store.go` — adjust comment on `DeleteFeedFollow`
- `internal/adapters/sqlc/feed_store.go` — translate the 0-row case
- `internal/app/feed.go` — `Unfollow` propagates the sentinel (current logic already does)
- `internal/adapters/cli/handlers_follow.go` — translate sentinel to friendly message in `Unfollow`
- `internal/app/fakes_test.go` — `fakeFeedStore.DeleteFeedFollow` returns `ErrNotFollowing` when no matching row
- `internal/app/feed_test.go` — new sub-test for the missing-row case

### Changes

**1. `sql/queries/feed_follows.sql`** — change the annotation on `DeleteFeedFollow`:

```sql
-- name: DeleteFeedFollow :execrows
DELETE FROM feed_follows
WHERE feed_id = $1 AND user_id = $2;
```

Run `sqlc generate` after this edit. The generated function signature changes from `... error` to `... (int64, error)` (the count of rows affected).

**2. `internal/domain/errors.go`** — append the sentinel:

```go
// ErrNotFollowing is returned by FeedStore.DeleteFeedFollow when the
// requested (user, feed) row does not exist. The use-case treats this
// as a user-facing error rather than a silent no-op.
ErrNotFollowing = errors.New("not following feed")
```

**3. `internal/ports/feed_store.go`** — rewrite the `DeleteFeedFollow` comment:

```go
// DeleteFeedFollow removes the (userID, feedID) follow row. Returns
// domain.ErrNotFollowing when no row matches; other errors are
// propagated as-is.
DeleteFeedFollow(ctx context.Context, userID, feedID uuid.UUID) error
```

**4. `internal/adapters/sqlc/feed_store.go`** — update `DeleteFeedFollow`:

```go
func (s *FeedStore) DeleteFeedFollow(ctx context.Context, userID, feedID uuid.UUID) error {
    rows, err := s.q.DeleteFeedFollow(ctx, database.DeleteFeedFollowParams{
        UserID: userID,
        FeedID: feedID,
    })
    if err != nil {
        return err
    }
    if rows == 0 {
        return domain.ErrNotFollowing
    }
    return nil
}
```

**5. `internal/app/feed.go`** — `Unfollow` is unchanged structurally; it already propagates whatever the store returns. Verify the existing implementation does `if err := s.store.DeleteFeedFollow(...); err != nil { return domain.Feed{}, err }` — no wrap needed (the sentinel is the whole point).

**6. `internal/adapters/cli/handlers_follow.go.Unfollow`** — translate the sentinel:

```go
feed, err := h.svc.Unfollow(ctx, user, args[0])
if err != nil {
    if errors.Is(err, domain.ErrNotFollowing) {
        return fmt.Errorf("not following %q", args[0])
    }
    return fmt.Errorf("failed to delete feed follow: %w", err)
}
```

**7. `internal/app/fakes_test.go.fakeFeedStore.DeleteFeedFollow`** — return the sentinel when no row matches:

```go
func (s *fakeFeedStore) DeleteFeedFollow(_ context.Context, userID, feedID uuid.UUID) error {
    if s.deleteFFErr != nil {
        return s.deleteFFErr
    }
    found := false
    out := s.follows[:0]
    for _, ff := range s.follows {
        if ff.UserID == userID && ff.FeedID == feedID {
            found = true
            continue
        }
        out = append(out, ff)
    }
    s.follows = out
    if !found {
        return domain.ErrNotFollowing
    }
    return nil
}
```

**8. `internal/app/feed_test.go.TestFeedService_Unfollow`** — add a sub-test:

```go
t.Run("unfollow with no matching row surfaces ErrNotFollowing", func(t *testing.T) {
    store := newFakeFeedStore()
    store.owners[owner.ID] = owner.Name
    store.feeds["https://example.com/rss"] = domain.Feed{ID: feedID, URL: "https://example.com/rss"}
    // note: store.follows is empty
    svc := NewFeedService(store, &fakeFetcher{}, &fakeClock{}, newFakeIDGen())

    _, err := svc.Unfollow(context.Background(), owner, "https://example.com/rss")
    if !errors.Is(err, domain.ErrNotFollowing) {
        t.Fatalf("err = %v, want ErrNotFollowing", err)
    }
})
```

### Verification

```bash
sqlc generate
go build ./...
go vet ./...
go test ./internal/... -count=1
```

Manual:

```bash
go run ./cmd/gator/ register polish-test
go run ./cmd/gator/ login polish-test
go run ./cmd/gator/ unfollow "https://does-not-exist.example.com"
# expect: Error: not following "https://does-not-exist.example.com"
go run ./cmd/gator/ reset
```

### Suggested commit message

```
fix: surface ErrNotFollowing when unfollow targets a non-existent row 🐛

Previously `gator unfollow <url>` printed "Feed unfollowed" even when
no follow row existed for the current user — a silent no-op the CLI
reported as success. Switch the DeleteFeedFollow sqlc annotation to
:execrows, translate the 0-row case to a new domain.ErrNotFollowing
sentinel at the adapter boundary, and surface a friendly "not
following <url>" message in the CLI handler.
```

---

## Commit 4 — C-B: agg per-tick log shows all five counters

### Goal

The current closing log line in `handlers_agg.tick`:

```go
log.Printf("Feed %s collected, %v posts found", result.Feed.Name, result.Found)
```

uses `Found` (the count of items in the feed XML), not `Inserted`. If half the items errored out, the log still claims "N posts found" as if everything was processed. Replace with a summary that names each counter.

### Files

- `internal/adapters/cli/handlers_agg.go`

### Changes

Replace lines around 75-82 in the `tick` function with:

```go
fmt.Fprintf(h.out, "Fetching feed %s\n", result.Feed.Name)
for _, e := range result.Errors {
    log.Printf("Failed to create post: %v", e)
}
log.Printf("Feed %s: %d found, %d inserted, %d dups, %d skipped, %d errors",
    result.Feed.Name,
    result.Found,
    result.Inserted,
    result.Dups,
    result.Skipped,
    len(result.Errors),
)
```

Drop the standalone `Skipped` summary line (it's now in the closing summary) — that's the line that currently reads:

```go
if result.Skipped > 0 {
    log.Printf("Skipped %d items with unparseable published date", result.Skipped)
}
```

Remove it. The unified closing line covers it.

### Verification

```bash
go build ./...
go vet ./...
go test ./internal/... -count=1
```

Manual (against a real feed):

```bash
go run ./cmd/gator/ register polish-test
go run ./cmd/gator/ login polish-test
go run ./cmd/gator/ addfeed boot "https://blog.boot.dev/index.xml"
go run ./cmd/gator/ agg 60s    # ctrl-C after one tick
# expect: Feed boot: N found, M inserted, K dups, J skipped, E errors
go run ./cmd/gator/ reset
```

### Suggested commit message

```
chore: agg per-tick log summary shows all 5 counters 🧹

The closing tick log used to print "N posts found" — confusing when
the feed had partial failures (some items inserted, some failed,
some duplicated, some skipped for unparseable pubDate). Switch to a
single summary line that names all five ScrapeResult counters so an
operator watching `gator agg` sees the truth at a glance.
```

---

## Commit 5 — C-D: validate non-empty inputs at the service layer

### Goal

Today, `gator addfeed "" ""` would land empty strings in the DB. Reject empty (or whitespace-only) names and URLs at the service layer with a clear error.

### Files

- `internal/app/feed.go` — `AddFeed`, `Follow`, `Unfollow` validate inputs
- `internal/app/user.go` — `Register`, `Login` validate name
- `internal/app/feed_test.go` — new sub-tests
- `internal/app/user_test.go` — new sub-tests

Validation lives at the service layer (not the CLI handler) because it's a domain rule, not a presentation concern. Use plain `fmt.Errorf` rather than a new sentinel — these are caller-side bugs that the user fixes by retyping the command, not branching error conditions the CLI needs to disambiguate.

### Changes

**1. `internal/app/feed.go`** — add validation at the top of three methods:

```go
import "strings"  // add to import block

func (s *FeedService) AddFeed(ctx context.Context, owner domain.User, name, url string) (domain.Feed, error) {
    name = strings.TrimSpace(name)
    url = strings.TrimSpace(url)
    if name == "" {
        return domain.Feed{}, fmt.Errorf("name cannot be empty")
    }
    if url == "" {
        return domain.Feed{}, fmt.Errorf("url cannot be empty")
    }
    // ... existing body ...
}

func (s *FeedService) Follow(ctx context.Context, owner domain.User, url string) (domain.FeedFollow, error) {
    url = strings.TrimSpace(url)
    if url == "" {
        return domain.FeedFollow{}, fmt.Errorf("url cannot be empty")
    }
    // ... existing body ...
}

func (s *FeedService) Unfollow(ctx context.Context, owner domain.User, url string) (domain.Feed, error) {
    url = strings.TrimSpace(url)
    if url == "" {
        return domain.Feed{}, fmt.Errorf("url cannot be empty")
    }
    // ... existing body ...
}
```

**2. `internal/app/user.go`** — same for Register and Login:

```go
import "strings"  // add to import block

func (s *UserService) Register(ctx context.Context, name string) (domain.User, error) {
    name = strings.TrimSpace(name)
    if name == "" {
        return domain.User{}, fmt.Errorf("name cannot be empty")
    }
    // ... existing body ...
}

func (s *UserService) Login(ctx context.Context, name string) (domain.User, error) {
    name = strings.TrimSpace(name)
    if name == "" {
        return domain.User{}, fmt.Errorf("name cannot be empty")
    }
    // ... existing body ...
}
```

**3. `internal/app/feed_test.go`** — add table tests for the rejection cases. One sub-test per method is enough:

```go
t.Run("AddFeed rejects empty name", func(t *testing.T) {
    svc := NewFeedService(newFakeFeedStore(), &fakeFetcher{}, &fakeClock{now: now}, newFakeIDGen())
    _, err := svc.AddFeed(context.Background(), owner, "   ", "https://example.com/rss")
    if err == nil || !strings.Contains(err.Error(), "name cannot be empty") {
        t.Errorf("err = %v, want 'name cannot be empty'", err)
    }
})
t.Run("AddFeed rejects empty url", func(t *testing.T) {
    svc := NewFeedService(newFakeFeedStore(), &fakeFetcher{}, &fakeClock{now: now}, newFakeIDGen())
    _, err := svc.AddFeed(context.Background(), owner, "Example", "  ")
    if err == nil || !strings.Contains(err.Error(), "url cannot be empty") {
        t.Errorf("err = %v, want 'url cannot be empty'", err)
    }
})
```

Add the equivalents for `Follow`, `Unfollow`.

**4. `internal/app/user_test.go`** — same:

```go
t.Run("Register rejects empty name", func(t *testing.T) {
    svc := NewUserService(newFakeUserStore(), &fakeClock{now: now}, newFakeIDGen(), &fakeSession{})
    _, err := svc.Register(context.Background(), "  ")
    if err == nil || !strings.Contains(err.Error(), "name cannot be empty") {
        t.Errorf("err = %v, want 'name cannot be empty'", err)
    }
})
t.Run("Login rejects empty name", func(t *testing.T) {
    svc := NewUserService(newFakeUserStore(), &fakeClock{now: now}, newFakeIDGen(), &fakeSession{})
    _, err := svc.Login(context.Background(), "")
    if err == nil || !strings.Contains(err.Error(), "name cannot be empty") {
        t.Errorf("err = %v, want 'name cannot be empty'", err)
    }
})
```

### Verification

```bash
go build ./...
go vet ./...
go test ./internal/... -count=1
```

Manual:

```bash
go run ./cmd/gator/ register "   "
# expect: Error: name cannot be empty
go run ./cmd/gator/ addfeed "" ""
# expect: Error: name cannot be empty (since name is checked first)
```

### Suggested commit message

```
feat: validate non-empty name and url inputs in user/feed services ✨

UserService.Register/Login and FeedService.AddFeed/Follow/Unfollow
now trim whitespace and reject empty values with "X cannot be empty"
errors. Plain fmt.Errorf rather than a sentinel — these are typos
the user retypes, not branching error conditions the CLI needs to
distinguish from other failure modes.

Each method has a sub-test for the rejection case.
```

---

## Commit 6 — D2: unit test the mapping helpers

### Goal

`isDuplicateKey` and `duplicateKeyConstraint` are tested only indirectly through end-to-end paths. Add a small direct test file that exercises them against synthetic `*pq.Error` values, so a future change to either helper (e.g. handling a new Postgres error code) breaks a fast unit test instead of a slow integration run.

### Files

- `internal/adapters/sqlc/mapping_test.go` (new)

### Changes

Create a new file:

```go
package sqlc

import (
    "errors"
    "fmt"
    "testing"

    "github.com/lib/pq"
)

func TestIsDuplicateKey(t *testing.T) {
    cases := []struct {
        name string
        err  error
        want bool
    }{
        {"nil", nil, false},
        {"non-pq error", errors.New("connection refused"), false},
        {"pq error with wrong code", &pq.Error{Code: "42P01"}, false},
        {"pq error with 23505", &pq.Error{Code: "23505"}, true},
        {"wrapped pq 23505", fmt.Errorf("create: %w", &pq.Error{Code: "23505"}), true},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            if got := isDuplicateKey(tc.err); got != tc.want {
                t.Errorf("isDuplicateKey(%v) = %v, want %v", tc.err, got, tc.want)
            }
        })
    }
}

func TestDuplicateKeyConstraint(t *testing.T) {
    cases := []struct {
        name string
        err  error
        want string
    }{
        {"nil", nil, ""},
        {"non-pq error", errors.New("connection refused"), ""},
        {"pq error with wrong code", &pq.Error{Code: "42P01", Constraint: "ignored"}, ""},
        {"feeds_url_key", &pq.Error{Code: "23505", Constraint: "feeds_url_key"}, "feeds_url_key"},
        {"feed_follows uniqueness", &pq.Error{Code: "23505", Constraint: "feed_follows_user_id_feed_id_key"}, "feed_follows_user_id_feed_id_key"},
        {"wrapped 23505", fmt.Errorf("insert: %w", &pq.Error{Code: "23505", Constraint: "users_name_key"}), "users_name_key"},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            if got := duplicateKeyConstraint(tc.err); got != tc.want {
                t.Errorf("duplicateKeyConstraint(%v) = %q, want %q", tc.err, got, tc.want)
            }
        })
    }
}
```

### Verification

```bash
go test ./internal/adapters/sqlc/... -v -count=1
```

Both tests should pass; the case count gives the helpers full branch coverage.

### Suggested commit message

```
test: cover isDuplicateKey and duplicateKeyConstraint helpers 🧪

Direct unit tests against synthetic *pq.Error values. Catches future
changes to either helper (new pg error codes, constraint-name parsing
changes) with a fast unit test instead of an integration run.
```

---

## Commit 7 — D1: unit tests for CLI handlers

### Goal

The CLI handlers are the largest untested surface in the repo. They contain usage-string parsing, sentinel → friendly-message translation, output formatting, and the `RequireLogin` middleware. The architecture (handlers take `io.Writer`; services are interfaces) makes them easy to test — stub the service, capture the buffer, assert on output and returned errors.

### Files

- `internal/adapters/cli/handlers_user_test.go` (new)
- `internal/adapters/cli/handlers_feed_test.go` (new)
- `internal/adapters/cli/handlers_follow_test.go` (new)
- `internal/adapters/cli/handlers_following_test.go` (new)
- `internal/adapters/cli/handlers_browse_test.go` (new)
- `internal/adapters/cli/handlers_agg_test.go` (new)
- `internal/adapters/cli/middleware_test.go` (new)
- `internal/adapters/cli/stubs_test.go` (new — shared stubs for the service interfaces)

### Implementation note: stub services, not the real `app.*` types

The handler structs hold concrete `*app.UserService` / `*app.FeedService` etc. (not interfaces). To test handlers without spinning up a real service + stores + clock, **refactor the handler structs to depend on minimal interfaces** declared in the cli package, satisfied by both `*app.UserService` and a stub.

For each handler, declare a `…Service` interface in the cli package — methods are exactly what the handler calls, no more:

```go
// In handlers_user.go (replace the struct field type)
type userService interface {
    Login(ctx context.Context, name string) (domain.User, error)
    Register(ctx context.Context, name string) (domain.User, error)
    ListUsers(ctx context.Context) ([]domain.User, error)
    Reset(ctx context.Context) error
    CurrentUserName() string
}

type UserHandlers struct {
    out io.Writer
    svc userService
}

func NewUserHandlers(out io.Writer, svc userService) *UserHandlers {
    return &UserHandlers{out: out, svc: svc}
}
```

The real `*app.UserService` already satisfies this set; no app-layer changes needed. Same for `feedService`, `browseService`, `scrapeService`. **Confirm with `go build ./...` after the type swap that `cmd/gator/main.go` still compiles** — passing the concrete service to a function that wants the interface is implicit-satisfaction, so no changes there either.

Once the interfaces exist, write a `stubs_test.go` with one minimal stub per service:

```go
package cli

import (
    "context"

    "github.com/farulivan/gator-go/internal/app"
    "github.com/farulivan/gator-go/internal/domain"
)

type stubUserService struct {
    loginUser     domain.User
    loginErr      error
    registerUser  domain.User
    registerErr   error
    listUsers     []domain.User
    listErr       error
    resetErr      error
    currentName   string
    // record calls
    loginArg      string
    registerArg   string
}

func (s *stubUserService) Login(_ context.Context, name string) (domain.User, error) {
    s.loginArg = name
    return s.loginUser, s.loginErr
}
// ... etc
```

Stubs for `feedService`, `browseService`, `scrapeService` follow the same pattern.

### Test coverage to add

**`handlers_user_test.go`** — for each of Login/Register/Users/Reset:
- Usage string returned when arg count is wrong.
- Happy path: service called with right args, expected output written to buffer.
- Error path: stubbed service error propagates with the wrapped message.
- For Register specifically: `ErrUserExists` translates to "username 'X' already exists".

**`handlers_feed_test.go`** — for AddFeed/ListFeeds:
- Usage strings.
- AddFeed: `ErrFeedExists` → "feed URL ... already exists; use `follow ...`" message.
- ListFeeds: empty list still prints "Feeds:" header.
- `printFeed` / `printFeedWithOwner` / `formatLastFetched` — exercise via the handler path (no need to test directly).

**`handlers_follow_test.go`** — Follow/Unfollow:
- Usage strings.
- Follow: `ErrAlreadyFollowing` → "already following ..." message.
- Unfollow: `ErrNotFollowing` → "not following ..." message (this lands in commit 3).

**`handlers_following_test.go`** — Following:
- Empty follows → "No feeds followed".
- Non-empty → "Feed follows for user: <name>" header followed by indented feed names.

**`handlers_browse_test.go`** — Browse:
- Default limit (no arg) → service receives 2.
- Explicit limit → service receives parsed value.
- Invalid limit → "invalid limit: ..." error.
- Empty result → "Found 0 posts for user X" with no items.

**`handlers_agg_test.go`** — Agg:
- Usage string when arg count wrong.
- Invalid duration → "invalid duration: ..." error.
- Hard-to-test: the actual loop. Lift `tick` into a directly-testable method if it isn't already exposed within the package — it already is at `internal/adapters/cli/handlers_agg.go:67` (`func (h *AggHandlers) tick(ctx context.Context) error`). Use the stub `scrapeService` to feed a `ScrapeResult`, capture the buffer, and assert the closing summary line shows all 5 counters (this also pins the change from commit 4).

**`middleware_test.go`** — RequireLogin:
- Empty session → returns `domain.ErrNotLoggedIn`; wrapped handler never invoked.
- Session names a user that doesn't exist → returns wrapped "failed to get user" error; wrapped handler never invoked.
- Happy path: wrapped handler invoked with the resolved user.

### Verification

```bash
go build ./...
go vet ./...
go test ./internal/adapters/cli/... -v -count=1
```

The CLI handler test count should jump by ~25 tests. Total project test count: ~50.

### Suggested commit message

```
test: add unit tests for CLI handlers and RequireLogin 🧪

Introduce minimal interfaces (userService, feedService, browseService,
scrapeService) so handlers can be tested against stubs without
constructing the real app services. The concrete *app.UserService etc.
satisfy the interfaces by Go's implicit-satisfaction rule, so the
composition root in cmd/gator/main.go is unchanged.

New tests cover:
- Usage strings for every verb (wrong-arity → "usage: ...")
- Sentinel → friendly-message translation: ErrFeedExists,
  ErrAlreadyFollowing, ErrNotFollowing, ErrUserExists.
- RequireLogin's two failure modes (empty session, user not found).
- AggHandlers.tick output: all 5 ScrapeResult counters present in
  the closing log line.

Test count: +~25.
```

---

## Commit 8 — D3: extend e2e harness to cover friendly duplicate-write messages

### Goal

The end-to-end harness in `e2e/e2e_test.go` runs the happy paths but never asserts on the new friendly error messages added in commits 1, 3 of `chore/post-refactor-followups` (B for ErrFeedExists / ErrAlreadyFollowing) and commit 3 of this branch (ErrNotFollowing). Add three assertions.

### Files

- `e2e/e2e_test.go`

### Changes

After the existing happy-path assertions (around line 70), add a "negative path" block:

```go
// Negative paths: each of the four unique-violation translations should
// surface a friendly message, not raw pq output.
if out := h.run(t, "addfeed", testFeedName, feedURL); !strings.Contains(out, "already exists") {
    // Second addfeed of the same URL — the message format is checked
    // through stderr in practice, but `run` only captures stdout. The
    // command exits non-zero, which run() treats as failure, so
    // restructure: capture both streams or check exit status.
    t.Logf("expected addfeed of duplicate URL to fail with 'already exists'; got stdout:\n%s", out)
}
```

**Caveat for implementation:** the existing `h.run` helper (`e2e_test.go:151-181`) fails the test on a non-zero exit. The friendly-error commands exit non-zero (they propagate the error to `log.Fatal`). To assert on them, either:

1. Add a `runExpectFail(t, args...)` helper that captures stderr, asserts a non-zero exit, and returns the captured stderr; or
2. Restructure the existing `run` to return `(stdout, stderr, exitCode)` and let each call site decide what's acceptable.

Option 1 is less invasive. Sketch:

```go
func (h *harness) runExpectFail(t *testing.T, args ...string) string {
    t.Helper()
    cmd := exec.Command(h.bin, args...)
    cmd.Env = append(os.Environ(), "HOME="+h.home)
    cmd.Dir = h.repoRoot
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    err := cmd.Run()
    if err == nil {
        t.Fatalf("gator %s: expected non-zero exit, got success\nstdout:\n%s",
            strings.Join(args, " "), stdout.String())
    }
    return stderr.String()
}
```

Then add to the test body, after the existing happy path:

```go
// Re-add already-added feed → "already exists"
if stderr := h.runExpectFail(t, "addfeed", testFeedName, feedURL); !strings.Contains(stderr, "already exists") {
    t.Fatalf("addfeed of duplicate URL: expected stderr to contain 'already exists'; got:\n%s", stderr)
}

// Re-follow already-followed feed → "already following"
if stderr := h.runExpectFail(t, "follow", feedURL); !strings.Contains(stderr, "already following") {
    t.Fatalf("follow of already-followed feed: expected 'already following'; got:\n%s", stderr)
}

// Unfollow a feed we don't follow → "not following"
const ghostURL = "https://does-not-exist.invalid/rss"
if stderr := h.runExpectFail(t, "unfollow", ghostURL); !strings.Contains(stderr, "not following") {
    t.Fatalf("unfollow of missing follow: expected 'not following'; got:\n%s", stderr)
}
```

### Verification

```bash
GATOR_E2E_DESTRUCTIVE=1 go test -tags=integration -v ./e2e/...
```

The harness should run through the happy path, then through the three negative paths, all green.

### Suggested commit message

```
test: extend e2e harness to cover friendly duplicate-write messages 🧪

Add three negative-path assertions to TestEndToEnd:
- addfeed of an existing URL → stderr contains "already exists"
- follow of an already-followed feed → stderr contains "already following"
- unfollow of a non-existent follow → stderr contains "not following"

Introduce a runExpectFail harness helper that captures stderr and
asserts a non-zero exit — required because the friendly-error commands
deliberately exit non-zero.
```

---

## Explicitly deferred (do NOT implement in this branch)

The review surfaced these items; each is intentionally left out of this plan with reasoning. Future planners can pick any of them up as a separate effort.

- **B1 — `feedFromCreateRow` ↔ `feedToDomain` near-duplication.** Two ~7-line projection functions that look identical but project different sqlc-generated row types. The duplication is a sqlc artifact (two structs with the same fields). Extracting a shared helper would push field-by-field parameters across the function boundary; readability is roughly a wash. Live with it; if the `feeds` schema gains a column, update both sites in lockstep.
- **C-C — Verb-name duplication in usage strings.** `handlers_user.go:28` says `"usage: login <name>"` while `cmd/gator/main.go:63` registers `"login"`. Cost of cleanup: thread the verb name through the `Handler` signature, which breaks the current func-type. Not worth the invasiveness for a string that rarely changes.
- **C-E — Unused `updated_at` columns.** Three of the four tables (`users`, `feed_follows`, `posts`) set `updated_at` only at INSERT. The columns are effectively `created_at_v2`. Cleanup options: (a) drop the columns in a goose migration; (b) actually use them in UPDATE queries. Either is a schema-level decision that warrants its own design conversation; both are migration-bearing and not part of "polish."
- **E1 — Structured logging.** Switch to `slog` with leveled output across the CLI and adapters. Reasonable for a production daemon; overkill for a CLI learning project. Note in the README roadmap if pursuing.
- **E2 — `db.Close()` error logging.** One-line addition; low value. Folded into a hypothetical "production hardening" branch instead of cluttering this plan.
- **E3 — `sql.DB` connection-pool tuning.** Defaults are fine for a CLI; matters for a long-running service.
- **E4 — Concurrent feed fetching in `agg`.** Already on the README roadmap. Larger design effort; sequential fetch is fine while feed count is small.
- **E5 — File-locking on `~/.gatorconfig.json`.** Theoretical race between two parallel `gator` processes calling `SetCurrentUser`. Not observed in practice; flag if the CLI ever grows a daemon mode.

---

## Critical files reference

Single source of truth for paths cited above:

- `internal/app/feed.go` — AddFeed, Follow, Unfollow (commits 1, 5)
- `internal/app/user.go` — Register, Login (commit 5)
- `internal/app/fakes_test.go` — fake store updates (commits 1, 3)
- `internal/app/feed_test.go`, `internal/app/user_test.go` — new sub-tests (commits 1, 3, 5)
- `internal/adapters/sqlc/feed_store.go` — DeleteFeedFollow translation (commit 3)
- `internal/adapters/sqlc/mapping.go` — isDuplicateKey simplification (commit 2)
- `internal/adapters/sqlc/mapping_test.go` (new) — direct helper tests (commit 6)
- `internal/adapters/cli/handlers_*.go` — interface refactor + handlers (commit 7)
- `internal/adapters/cli/*_test.go` (new) — CLI handler tests (commit 7)
- `internal/adapters/cli/handlers_agg.go` — log line summary (commit 4)
- `internal/adapters/cli/handlers_follow.go` — Unfollow message (commit 3)
- `internal/adapters/cli/middleware.go` — touched only by the test in commit 7
- `internal/ports/feed_store.go` — DeleteFeedFollow doc comment (commit 3)
- `internal/domain/errors.go` — ErrNotFollowing (commit 3)
- `sql/queries/feed_follows.sql` — :execrows annotation (commit 3)
- `internal/database/feed_follows.sql.go` — regenerated by sqlc generate (commit 3)
- `e2e/e2e_test.go` — runExpectFail helper + 3 assertions (commit 8)

## End-of-branch verification

After all eight commits land:

```bash
go build ./...
go vet ./...
go test ./internal/... -count=1
grep -rE 'database/sql|net/http|encoding/xml|internal/database' internal/domain internal/ports internal/app
# expected: no output
grep -rE 'github.com/lib/pq' internal/domain internal/ports internal/adapters/cli internal/app cmd
# expected: only cmd/gator/main.go (the blank import)
grep -rE 'github.com/lib/pq' internal/adapters/sqlc
# expected: mapping.go (and mapping_test.go after commit 6)
GATOR_E2E_DESTRUCTIVE=1 go test -tags=integration -v ./e2e/...
# expected: TestEndToEnd green, including the three new negative-path assertions
```

## Pull request

Open a PR titled:

```
chore: polish & hardening follow-ups for hexagonal architecture
```

Target branch: whichever holds the C1/C2 follow-ups (likely `refactor/hexagonal-architecture` if it hasn't merged to `main` yet, otherwise `main`).

Description sketch:

```markdown
## Summary

Follow-ups from the senior-architect review of the hexagonal-architecture
branch. None change architecture; all are small. Eight commits:

- 2x refactor: AddFeed plumbing polish + remove dead substring fallback
- 1x fix: surface ErrNotFollowing on missing-row unfollow
- 1x chore: agg per-tick log summary shows all 5 counters
- 1x feat: validate non-empty inputs in services
- 3x test: mapping helpers, CLI handlers, e2e duplicate-message coverage

## What this branch deliberately does NOT do

See `docs/polish-hardening-plan.md` "Explicitly deferred" — five items
intentionally left for follow-up branches (B1 mapping duplication, C-C
verb-name duplication, C-E unused updated_at columns, E1-E5 production
hardening).

## Test plan

- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] `go test ./internal/... -count=1` — now includes CLI handler tests
- [ ] `GATOR_E2E_DESTRUCTIVE=1 go test -tags=integration -v ./e2e/...`
- [ ] Manual: addfeed twice, follow twice, unfollow ghost URL — each surfaces a friendly message instead of raw pq output.
```
