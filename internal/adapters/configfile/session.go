// Package configfile is the production adapter for the ports.Session seam.
// It is a thin shim over internal/config so the rest of the codebase can
// keep `current_user_name` flowing through ~/.gatorconfig.json without
// importing the config package directly. Read() stays in main; this
// adapter only wraps an already-loaded *config.Config.
package configfile

import (
	"github.com/farulivan/gator-go/internal/config"
	"github.com/farulivan/gator-go/internal/ports"
)

// Session wraps a *config.Config so it satisfies ports.Session.
type Session struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Session {
	return &Session{cfg: cfg}
}

var _ ports.Session = (*Session)(nil)

func (s *Session) CurrentUserName() string {
	return s.cfg.CurrentUserName
}

func (s *Session) SetCurrentUser(name string) error {
	return s.cfg.SetUser(name)
}
