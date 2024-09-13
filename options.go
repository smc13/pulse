package pulse

import "log/slog"

type PulseOption func(*Pulser) error
type ListenerOption func(*Listener)

// WithSlog sets the slog.Logger for the Pulser
// by default slog.Default() is used (in future versions this will be changed to a discardHandler)
// ! TODO: replace slog.Default() with a discardHandler when available in go
func WithSlog(s *slog.Logger) PulseOption {
	return func(p *Pulser) error {
		p.slog = s
		return nil
	}
}

// WithName sets the name of the listener, useful for debugging
// defaults to an integer id
func WithName(name string) ListenerOption {
	return func(l *Listener) {
		l.name = name
	}
}
