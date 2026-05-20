// Package cli is the driving adapter that turns argv into use-case calls
// and use-case results into stdout. It owns argv parsing, dispatch, and
// output formatting; nothing inside `internal/app` or `internal/domain`
// imports it.
package cli

import "context"

// Handler is the signature every CLI verb registers under. The command
// name is consumed by the Router for dispatch and is intentionally not
// part of this signature; handlers should hardcode their usage strings.
type Handler func(ctx context.Context, args []string) error
