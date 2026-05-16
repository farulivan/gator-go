package cli

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Router maps a command name to its Handler and writes all output to a
// single io.Writer (default os.Stdout). Tests can construct a Router with
// a *bytes.Buffer to capture output, which is why handlers in this
// package use fmt.Fprintf(r.Out(), …) rather than fmt.Printf.
type Router struct {
	out      io.Writer
	handlers map[string]Handler
}

// NewRouter returns a Router writing to the given io.Writer. A nil writer
// falls back to os.Stdout.
func NewRouter(out io.Writer) *Router {
	if out == nil {
		out = os.Stdout
	}
	return &Router{out: out, handlers: make(map[string]Handler)}
}

// Out exposes the writer so handler structs can format output through the
// same sink the Router was constructed with.
func (r *Router) Out() io.Writer { return r.out }

// Register binds a Handler to a command name. The latest registration
// wins; double-registration is a programming error so we don't guard
// against it.
func (r *Router) Register(name string, h Handler) {
	r.handlers[name] = h
}

// Has reports whether a Handler is registered for the given command name.
// Used during the Group 3 transitional period to decide between the new
// Router and the legacy `commands` registry.
func (r *Router) Has(name string) bool {
	_, ok := r.handlers[name]
	return ok
}

// Run dispatches a single command. args[0] is the command name; args[1:]
// is forwarded to the Handler.
func (r *Router) Run(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: gator <command> [args...]")
	}
	name, rest := args[0], args[1:]
	h, ok := r.handlers[name]
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}
	return h(ctx, rest)
}
