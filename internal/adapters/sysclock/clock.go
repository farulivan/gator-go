// Package sysclock is the production adapter for the ports.Clock seam. It
// returns wall-clock UTC time. Tests inject a fake clock instead.
package sysclock

import (
	"time"

	"github.com/farulivan/gator-go/internal/ports"
)

// Clock implements ports.Clock by returning time.Now().UTC(). The struct is
// stateless; a value receiver is fine.
type Clock struct{}

func New() Clock { return Clock{} }

var _ ports.Clock = Clock{}

func (Clock) Now() time.Time { return time.Now().UTC() }
