package cast

import (
	"errors"
	"testing"
	"time"
)

type failingConnection chan error

func (c failingConnection) Send() chan<- *Message    { return nil }
func (c failingConnection) Receive() <-chan *Message { return nil }
func (c failingConnection) Error() <-chan error      { return c }

func TestMessageChannel(t *testing.T) {
	testMessageChannel(t, "simple message channel", func(buffer int, timeout time.Duration) (Connection, []Connection) {
		mc := make(MessageChannel, buffer)

		var c Connection = mc
		if timeout != 0 {
			c = NewTimeoutConnection(c, timeout)
		}

		return c, []Connection{c}
	})
}

func TestInProcChannelBehaviorLocal(t *testing.T) {
	testMessageChannel(t, "in-proc, local", func(buffer int, timeout time.Duration) (Connection, []Connection) {
		local, remote := NewInProcConnection()

		if buffer != 0 {
			local = NewBufferedConnection(local, buffer)
		}

		if timeout != 0 {
			local = NewTimeoutConnection(local, timeout)
		}

		return local, []Connection{remote}
	})
}

func TestInProcChannelBehaviorRemote(t *testing.T) {
	testMessageChannel(t, "in-proc, remote", func(buffer int, timeout time.Duration) (Connection, []Connection) {
		local, remote := NewInProcConnection()

		if buffer != 0 {
			remote = NewBufferedConnection(remote, buffer)
		}

		if timeout != 0 {
			remote = NewTimeoutConnection(remote, timeout)
		}

		return remote, []Connection{local}
	})
}

func TestBufferedConnectionForwardsError(t *testing.T) {
	fc := make(failingConnection)
	err := errors.New("test error")
	go func() { fc <- err }()
	bc := NewBufferedConnection(fc, 0)
	select {
	case errBack := <-bc.Error():
		if errBack != err {
			t.Error("failed to forward error")
		}
	case <-time.After(120 * time.Millisecond):
		t.Error("failed to forward error")
	}
}

func TestTimeoutConnectionForwardsError(t *testing.T) {
	fc := make(failingConnection)
	err := errors.New("test error")
	go func() { fc <- err }()
	bc := NewTimeoutConnection(fc, 0)
	select {
	case errBack := <-bc.Error():
		if errBack != err {
			t.Error("failed to forward error")
		}
	case <-time.After(120 * time.Millisecond):
		t.Error("failed to forward error")
	}
}
