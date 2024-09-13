package pulse

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
)

type CloseFunc func()
type SubKey[T any] struct{}
type Handler[T any] func(context.Context, T) error
type Listener struct {
	name string
	h    any
}

type NamedEvent interface {
	Kind() string
}

type Pulser struct {
	idGen int
	subs  map[any][]any

	mx   sync.RWMutex
	slog *slog.Logger
}

// NewPulser creates a new Pulser.
// It accepts a list of PulseOption functions to configure the Pulser and returns an error if any of the options fail
func NewPulser(opts ...PulseOption) (*Pulser, error) {
	pulser := &Pulser{
		slog: slog.Default(),
	}

	for _, opt := range opts {
		if err := opt(pulser); err != nil {
			return nil, err
		}
	}

	return pulser, nil
}

// Emit sends a pulse to all listeners asynchronously.
// Listener errors are ignored, use PulseSync to get the first error that occurs
func Emit[T any](p *Pulser, ctx context.Context, e T) error {
	p.mx.RLock()
	defer p.mx.RUnlock()

	subs := p.subs[SubKey[T]{}]
	for _, l := range subs {
		go func(l Listener) {
			if err := l.h.(Handler[T])(ctx, e); err != nil {
				p.slog.ErrorContext(ctx, "error emitting pulse", slog.String("error", err.Error()), slog.String("listener", l.name))
			}
		}(l.(Listener))
	}

	return nil
}

// EmitSync sends a pulse to all listeners synchronously.
// Listeners are called in the order they were added and the first error that occurs will be returned
func EmitSync[T any](p *Pulser, ctx context.Context, e T) error {
	p.mx.RLock()
	defer p.mx.RUnlock()

	subs := p.subs[SubKey[T]{}]
	for _, l := range subs {
		if err := l.(Listener).h.(Handler[any])(ctx, e); err != nil {
			p.slog.ErrorContext(ctx, "error emitting pulse", slog.String("error", err.Error()), slog.String("listener", l.(Listener).name))
			return err
		}
	}

	return nil
}

// On create a new listener for the event that will call the listener function when the event is pulsed
// It returns a CloseFunc that should be called when the listener is no longer needed
func On[T any](p *Pulser, handler Handler[T], opts ...ListenerOption) CloseFunc {
	p.mx.Lock()
	defer p.mx.Unlock()

	if p.subs == nil {
		p.subs = make(map[any][]any)
	}

	key := SubKey[T]{}
	name := p.idGen
	p.idGen++

	listener := &Listener{h: handler, name: fmt.Sprintf("%d", name)}
	for _, opt := range opts {
		opt(listener)
	}

	p.subs[key] = append(p.subs[key], listener)
	off := func() {
		p.mx.Lock()
		defer p.mx.Unlock()

		subs := p.subs[key]
		for i, l := range subs {
			l, ok := l.(Listener)
			if !ok {
				continue
			}

			if l.name == listener.name {
				p.subs[key] = slices.Delete(subs, i, i+1)
				break
			}
		}
	}

	return off
}

// OnChan create a new listener for the event that will send the event to the channel
// It returns an unbuffered channel and a CloseFunc that should be called when the listener is no longer needed
func OnChan[T any](p *Pulser) (chan<- T, CloseFunc) {
	c := make(chan T)
	off := On(p, func(_ context.Context, t T) error {
		c <- t
		return nil
	})

	return c, func() {
		off()
		close(c)
	}
}
