package exports

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/smc13/pulse"
)

func WriteSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

func EventsToSSE(ctx context.Context, pulser *pulse.Pulser, w http.ResponseWriter, events ...string) error {
	listener, off := pulse.OnAnyChan(pulser)

	go func() {
		for {
			select {
			case <-ctx.Done():
				off()
				return
			case e := <-listener:
				kind := e.Kind()
				valid := slices.Contains(events, kind)

				if valid || len(events) == 0 {
					encoded, err := json.Marshal(e)
					if err != nil {
						continue
					}

					w.Write([]byte("event: " + kind + "\n"))
					w.Write([]byte("data: " + string(encoded) + "\n\n"))
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
			}
		}
	}()

	return nil
}
