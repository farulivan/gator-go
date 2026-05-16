package ports

import "time"

// Clock is the seam through which use-cases get the current time. The
// production adapter (sysclock) returns time.Now().UTC(); tests can inject
// a fake to assert on persisted timestamps.
type Clock interface {
	Now() time.Time
}
