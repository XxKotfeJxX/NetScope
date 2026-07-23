package diagnostics

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHubSubscribePublishAndCleanup(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	runID := uuid.New()
	events, unsubscribe := hub.Subscribe(runID)

	hub.Publish(runID, RunEvent{Type: EventRunStarted, RunID: runID})
	select {
	case event := <-events:
		if event.Type != EventRunStarted {
			t.Fatalf("event type = %q, want %q", event.Type, EventRunStarted)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	unsubscribe()
	if _, open := <-events; open {
		t.Fatal("subscription channel remains open")
	}
	unsubscribe()
}
