package pulse

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Pulser struct {
	logger *slog.Logger
	mx     sync.Mutex

	listeners    map[string][]*Listener[Event]
	anyListeners []*Listener[Event]
}

func NewPulser(opts ...PulseOption) (*Pulser, error) {
	pulser := &Pulser{
		logger:       slog.New(discardHandler{}),
		listeners:    make(map[string][]*Listener[Event]),
		anyListeners: make([]*Listener[Event], 0),
	}

	for _, opt := range opts {
		if err := opt(pulser); err != nil {
			return nil, err
		}
	}

	return pulser, nil
}

func (p *Pulser) createListener(e Event, handler Handler[Event]) *Listener[Event] {
	kind := "*"
	if e != nil {
		kind = e.Kind()
	}

	idx := len(p.listeners[kind])

	return &Listener[Event]{
		id:      fmt.Sprintf("%s:%d", kind, idx),
		mx:      sync.Mutex{},
		handler: handler,
	}
}

func (p *Pulser) addListener(e Event, handler Handler[Event]) *Listener[Event] {
	p.mx.Lock()
	defer p.mx.Unlock()

	kind := e.Kind()

	// ensure map and slice exist
	if _, ok := p.listeners[kind]; !ok {
		p.listeners[kind] = make([]*Listener[Event], 0)
	}

	listener := p.createListener(e, handler)
	p.listeners[kind] = append(p.listeners[kind], listener)
	return listener
}

func (p *Pulser) addAnyListener(handler Handler[Event]) *Listener[Event] {
	p.mx.Lock()
	defer p.mx.Unlock()

	listener := p.createListener(nil, handler)
	p.anyListeners = append(p.anyListeners, listener)

	p.logger.Debug("global event listener added", slog.String("id", listener.id))
	return listener
}

func (p *Pulser) removeListener(event Event, listener *Listener[Event]) {
	p.mx.Lock()
	defer p.mx.Unlock()

	kind := event.Kind()
	listeners := p.listeners[kind]

	for i, l := range listeners {
		if l.id == listener.id {
			p.listeners[kind] = append(listeners[:i], listeners[i+1:]...)
			return
		}
	}

	p.logger.Warn("listener not found", slog.String("id", listener.id))
}

func (p *Pulser) removeAnyListener(listener *Listener[Event]) {
	p.mx.Lock()
	defer p.mx.Unlock()

	for i, l := range p.anyListeners {
		if l.id == listener.id {
			p.anyListeners = append(p.anyListeners[:i], p.anyListeners[i+1:]...)
			return
		}
	}
}

// On registers the handler for the event type
// Calling the returned function will remove the listener
func On[T Event](pulser *Pulser, handler Handler[T]) func() {
	var t T
	listener := pulser.addListener(t, func(e Event) {
		if event, ok := e.(T); ok {
			handler(event)
		}
	})

	return func() { pulser.removeListener(t, listener) }
}

// OnAny registers the handler to accept any event
// Calling the returned function will remove the listener
func OnAny(pulser *Pulser, handler Handler[Event]) func() {
	listener := pulser.addAnyListener(handler)
	return func() { pulser.removeAnyListener(listener) }
}

// OnChan sends events through a channel
// calling the returned function will close the channel and remove the listener
func OnChan[T Event](pulser *Pulser) (chan T, func()) {
	ch := make(chan T, 10)
	off := On(pulser, func(event T) { ch <- event })

	return ch, func() {
		off()
		close(ch)
	}
}

func OnAnyChan(pulser *Pulser) (chan Event, func()) {
	ch := make(chan Event, 10)
	off := OnAny(pulser, func(event Event) { ch <- event })

	return ch, func() {
		off()
		close(ch)
	}
}

func (p *Pulser) Pulse(ctx context.Context, event Event) {
	p.mx.Lock()
	defer p.mx.Unlock()

	kind := event.Kind()

	listeners := p.listeners[kind]
	listeners = append(listeners, p.anyListeners...)

	p.logger.InfoContext(ctx, "pulsing event", slog.String("kind", kind), slog.Int("listeners", len(listeners)))

	for _, l := range listeners {
		p.logger.DebugContext(ctx, "pulsing event to listener", slog.String("id", l.id), slog.String("kind", kind))
		go l.Handle(event)
	}
}
