package cast

import (
	"testing"
	"time"
)

func TestDoesNotBlockInitially(t *testing.T) {
	n := NewNode(0, 0)

	done := make(chan struct{})
	go func() {
		n.Send() <- Message{}
		n.Send() <- Message{}
		n.Send() <- Message{}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(120 * time.Millisecond):
		t.Error("initial blocking")
	}
}
