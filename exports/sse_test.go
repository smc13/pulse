package exports_test

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/smc13/pulse"
	"github.com/smc13/pulse/exports"
	"github.com/stretchr/testify/assert"
)

type emitEvent struct {
	User string
}

func (e emitEvent) Kind() string { return "test" }

func TestEventsToSSE(t *testing.T) {
	pulser, _ := pulse.NewPulser(pulse.WithSlog(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	ctx, cancel := context.WithCancel(context.Background())

	// fake a http response
	w := httptest.NewRecorder()

	// call the function
	exports.EventsToSSE(ctx, pulser, w)

	// emit some events
	pulser.Pulse(ctx, emitEvent{User: "sam"})
	pulser.Pulse(ctx, emitEvent{User: "rose"})

	time.Sleep(500 * time.Millisecond)
	cancel() // fake the request ending

	expected := `event: test
data: {"User":"sam"}

event: test
data: {"User":"rose"}

`

	assert.Equal(t, expected, w.Body.String())
}
