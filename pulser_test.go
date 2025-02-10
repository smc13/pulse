package pulse

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestPulser() *Pulser {
	pulser, _ := NewPulser(WithSlog(slog.New(slog.NewTextHandler(
		os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	return pulser
}

func TestNewPulser(t *testing.T) {
	tests := []struct {
		name    string
		opts    []PulseOption
		want    *Pulser
		wantErr bool
	}{
		{
			name:    "no options",
			opts:    nil,
			want:    &Pulser{},
			wantErr: false,
		},
		{
			name: "returns an error",
			opts: []PulseOption{
				func(p *Pulser) error { return assert.AnError },
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "doesnt error",
			opts: []PulseOption{
				func(p *Pulser) error { return nil },
			},
			want:    &Pulser{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPulser(tt.opts...)
			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type TestEvent struct{}

func (e TestEvent) Kind() string { return "test" }

type DupeTestEvent struct {
	a string
}

func (e DupeTestEvent) Kind() string { return "test" }

func TestPulser_On(t *testing.T) {
	pulser := newTestPulser()
	recieveCh := make(chan struct{})

	On(pulser, func(e TestEvent) {
		t.Log("recieved event")
		recieveCh <- struct{}{}
	})

	pulser.Pulse(context.Background(), TestEvent{})

	select {
	case <-recieveCh:
		break
	case <-time.After(1 * time.Second):
		t.Error("timeout")
	}
}

func TestPulser_On_WrongType(t *testing.T) {
	pulser := newTestPulser()
	recieveCh := make(chan struct{})

	On(pulser, func(e DupeTestEvent) {
		t.Log("recieved event")
		recieveCh <- struct{}{}
	})

	pulser.Pulse(context.Background(), TestEvent{})

	select {
	case <-recieveCh:
		t.Error("timeout")
	case <-time.After(1 * time.Second):
		break
	}
}

func TestPulser_OnAny(t *testing.T) {
	pulser := newTestPulser()
	recieveCh := make(chan struct{})

	OnAny(pulser, func(e Event) {
		t.Log("recieved event")
		recieveCh <- struct{}{}
	})

	pulser.Pulse(context.Background(), TestEvent{})

	select {
	case <-recieveCh:
		break
	case <-time.After(1 * time.Second):
		t.Error("timeout")
	}
}

func TestPulser_OnChan(t *testing.T) {
	pulser := newTestPulser()
	recieveCh, off := OnChan[TestEvent](pulser)

	pulser.Pulse(context.Background(), TestEvent{})

	select {
	case <-recieveCh:
		t.Log("recieved event")
		break
	case <-time.After(1 * time.Second):
		t.Error("timeout")
	}

	off()
}

func TestPulser_OnAnyChan(t *testing.T) {
	pulser := newTestPulser()
	recieveCh, off := OnAnyChan(pulser)

	pulser.Pulse(context.Background(), TestEvent{})
	pulser.Pulse(context.Background(), DupeTestEvent{})

	select {
	case <-recieveCh:
		t.Log("recieved event")
		break
	case <-time.After(1 * time.Second):
		t.Error("timeout")
	}

	off()
}
