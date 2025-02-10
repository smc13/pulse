package pulse

import "log/slog"

type PulseOption func(*Pulser) error

// WithSlog sets the slog.Logger for the Pulser
// the default handler will discard all logs
func WithSlog(s *slog.Logger) PulseOption {
	return func(p *Pulser) error {
		p.logger = s
		return nil
	}
}
