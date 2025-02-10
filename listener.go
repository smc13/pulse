package pulse

import "sync"

type Handler[T Event] func(e T)
type Listener[T Event] struct {
	id      string
	mx      sync.Mutex
	handler Handler[T]
}

func (l *Listener[T]) Handle(e T) {
	l.mx.Lock()
	defer l.mx.Unlock()

	l.handler(e)
}
