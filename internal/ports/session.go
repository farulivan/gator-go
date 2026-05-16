package ports

// Session is the cross-cut the CLI uses to carry the logged-in user's name
// across CLI invocations. Only UserService (to call SetCurrentUser on
// Login/Register success) and the cli.Router middleware (to read
// CurrentUserName for `requireLogin`) consume this port; Feed/Browse/Scrape
// services accept a domain.User explicitly and never touch Session — see
// plan decision 6.
type Session interface {
	// CurrentUserName returns the empty string if no user is logged in.
	CurrentUserName() string

	// SetCurrentUser persists the new current user; returns the underlying
	// write error (e.g. failure to write ~/.gatorconfig.json).
	SetCurrentUser(name string) error
}
