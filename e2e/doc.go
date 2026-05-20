// Package e2e drives the gator CLI end-to-end against a real Postgres and a
// real RSS feed. It exists so every commit group in the hexagonal refactor
// can be verified the same way: a single `go test` invocation walks every
// registered command in main.go's dispatch table.
//
// # Usage
//
//	GATOR_E2E_DESTRUCTIVE=1 go test -tags=integration ./e2e/...
//
// # Why a build tag
//
// Without `-tags=integration` the test file is invisible to the toolchain,
// so `go build ./...`, `go vet ./...`, and `go test ./...` keep passing
// without a database or network. That keeps the per-group build smoke fast.
// This file (doc.go) carries no build tag so the package itself stays
// listable by `go list` and the language server, even with the tag absent.
//
// # Why GATOR_E2E_DESTRUCTIVE
//
// The test always begins with `gator reset`, which DELETEs every row in the
// users table (and cascades to feeds/follows/posts). Requiring an explicit
// env var prevents an accidental wipe of a dev database that happens to be
// pointed at by ~/.gatorconfig.json.
//
// # Sandboxing
//
// The test creates a fresh $HOME under t.TempDir() and writes its own
// .gatorconfig.json there, reusing the developer's db_url but starting
// with an empty current_user_name. The real ~/.gatorconfig.json is never
// touched.
//
// # Binary path auto-detect
//
// The composition root moves from ./ to ./cmd/gator/ in commit group 6 of
// the hexagonal refactor. The harness probes for ./cmd/gator/main.go and
// falls back to ./, so the same test runs unchanged before and after the
// move.
package e2e
