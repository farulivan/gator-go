// Package uuidgen is the production adapter for the ports.IDGen seam. It
// wraps github.com/google/uuid's New() so use-cases can request a fresh
// UUID without importing the third-party package directly.
package uuidgen

import (
	"github.com/farulivan/gator-go/internal/ports"
	"github.com/google/uuid"
)

// Generator is a stateless adapter producing v4 UUIDs.
type Generator struct{}

func New() Generator { return Generator{} }

var _ ports.IDGen = Generator{}

func (Generator) NewID() uuid.UUID { return uuid.New() }
