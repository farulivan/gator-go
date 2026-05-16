//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	defaultFeedURL = "https://blog.boot.dev/index.xml"
	testFeedName   = "example"
	testUser       = "e2etest"
	aggDeadline    = 12 * time.Second
)

// TestEndToEnd walks every command registered in main.go's dispatch table:
// reset, register, users, login, addfeed, feeds, following, unfollow,
// follow, agg, browse. Always starts with reset for determinism.
func TestEndToEnd(t *testing.T) {
	if os.Getenv("GATOR_E2E_DESTRUCTIVE") == "" {
		t.Skip("destructive integration test (calls `gator reset`); set GATOR_E2E_DESTRUCTIVE=1 to run")
	}

	feedURL := os.Getenv("GATOR_E2E_FEED_URL")
	if feedURL == "" {
		feedURL = defaultFeedURL
	}

	h := newHarness(t)

	// Always wipe first; the rest of the test asserts on counts and
	// containment, so the DB must start empty.
	h.run(t, "reset")

	// register sets the new user as current; users marks them; login is a
	// no-op idempotency check.
	h.run(t, "register", testUser)
	if out := h.run(t, "users"); !strings.Contains(out, testUser+" (current)") {
		t.Fatalf("users: expected %q marked current; got:\n%s", testUser, out)
	}
	h.run(t, "login", testUser)

	// addfeed implicitly follows; feeds and following both list it.
	h.run(t, "addfeed", testFeedName, feedURL)
	if out := h.run(t, "feeds"); !strings.Contains(out, feedURL) {
		t.Fatalf("feeds missing %q; got:\n%s", feedURL, out)
	}
	if out := h.run(t, "following"); !strings.Contains(out, "- "+testFeedName) {
		t.Fatalf("following missing %q; got:\n%s", testFeedName, out)
	}

	// unfollow / re-follow round-trip.
	h.run(t, "unfollow", feedURL)
	if out := h.run(t, "following"); strings.Contains(out, "- "+testFeedName) {
		t.Fatalf("following still lists %q after unfollow; got:\n%s", testFeedName, out)
	}
	h.run(t, "follow", feedURL)
	if out := h.run(t, "following"); !strings.Contains(out, "- "+testFeedName) {
		t.Fatalf("following missing %q after re-follow; got:\n%s", testFeedName, out)
	}

	// agg's loop fires the first scrape immediately, then waits for the
	// ticker. We give it aggDeadline to finish the HTTP fetch and inserts,
	// then SIGKILL via the cancelled context.
	h.runTimed(t, aggDeadline, "agg", "60s")

	// browse must list at least one post ingested by the agg tick above.
	if out := h.run(t, "browse", "5"); !strings.Contains(out, "from "+testFeedName) {
		t.Fatalf("browse has no posts from %q feed; got:\n%s", testFeedName, out)
	}
}

// harness owns the sandboxed HOME, the prebuilt gator binary, and the
// repoRoot used as the binary's working directory.
type harness struct {
	bin      string
	home     string
	repoRoot string
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	repoRoot := mustRepoRoot(t)

	// Reuse the developer's existing db_url; the schema is already applied
	// against that database, so we don't need a separate test DB to migrate.
	realCfgPath := filepath.Join(os.Getenv("HOME"), ".gatorconfig.json")
	real, err := os.ReadFile(realCfgPath)
	if err != nil {
		t.Fatalf("read %s (needed for db_url): %v", realCfgPath, err)
	}
	var existing struct {
		DBURL string `json:"db_url"`
	}
	if err := json.Unmarshal(real, &existing); err != nil {
		t.Fatalf("parse %s: %v", realCfgPath, err)
	}
	if existing.DBURL == "" {
		t.Fatalf("%s: db_url is empty", realCfgPath)
	}

	// Sandbox $HOME so the test's register/login mutations stay out of the
	// developer's real config.
	home := t.TempDir()
	cfg, _ := json.Marshal(map[string]string{
		"db_url":            existing.DBURL,
		"current_user_name": "",
	})
	if err := os.WriteFile(filepath.Join(home, ".gatorconfig.json"), cfg, 0600); err != nil {
		t.Fatal(err)
	}

	// Build once: avoids `go run` orphaning the agg subprocess when
	// CommandContext SIGKILLs the parent, and is far faster than recompiling
	// per command.
	pkgPath := "."
	if _, err := os.Stat(filepath.Join(repoRoot, "cmd/gator/main.go")); err == nil {
		pkgPath = "./cmd/gator/"
	}
	bin := filepath.Join(t.TempDir(), "gator")
	build := exec.Command("go", "build", "-o", bin, pkgPath)
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build %s: %v\n%s", pkgPath, err, out)
	}

	return &harness{bin: bin, home: home, repoRoot: repoRoot}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// run invokes gator with the given args, fails on non-zero exit, and
// returns captured stdout for assertions.
func (h *harness) run(t *testing.T, args ...string) string {
	t.Helper()
	return h.runCtx(t, context.Background(), false, args...)
}

// runTimed invokes gator with a deadline, intended for `agg` and any other
// long-running command we expect to be killed. A non-zero exit (from
// SIGKILL) is NOT a failure.
func (h *harness) runTimed(t *testing.T, deadline time.Duration, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()
	return h.runCtx(t, ctx, true, args...)
}

func (h *harness) runCtx(t *testing.T, ctx context.Context, allowKilled bool, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, h.bin, args...)
	cmd.Env = append(os.Environ(), "HOME="+h.home)
	cmd.Dir = h.repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, &testLogger{t: t})
	cmd.Stderr = io.MultiWriter(&stderr, &testLogger{t: t, prefix: "stderr: "})

	if err := cmd.Run(); err != nil && !allowKilled {
		t.Fatalf("gator %s: %v\nstdout:\n%s\nstderr:\n%s",
			strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// testLogger streams subprocess output to t.Log so it interleaves
// chronologically with the assertion lines under `go test -v`.
type testLogger struct {
	t      *testing.T
	prefix string
}

func (l *testLogger) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if line != "" {
			l.t.Log(l.prefix + line)
		}
	}
	return len(p), nil
}
