package ports

import "github.com/google/uuid"

// IDGen is the seam through which use-cases mint new UUIDs. The production
// adapter (uuidgen) wraps `uuid.New()`; tests can inject a deterministic
// generator.
type IDGen interface {
	NewID() uuid.UUID
}
