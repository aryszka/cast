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

func makeMessageChannel(buffer int, timeout time.Duration) (Connection, []Connection) {
	mc := make(MessageChannel, buffer)

	var c Connection = mc
	if timeout != 0 {
		c = NewTimeoutConnection(c, timeout)
	}

	return c, []Connection{c}
}

func makeInProcChannelLocal(buffer int, timeout time.Duration) (Connection, []Connection) {
	local, remote := NewInProcConnection()

	if buffer != 0 {
		local = NewBufferedConnection(local, buffer)
	}

	if timeout != 0 {
		local = NewTimeoutConnection(local, timeout)
	}

	return local, []Connection{remote}
}

func makeInProcChannelRemote(buffer int, timeout time.Duration) (Connection, []Connection) {
	local, remote := NewInProcConnection()

	if buffer != 0 {
		remote = NewBufferedConnection(remote, buffer)
	}

	if timeout != 0 {
		remote = NewTimeoutConnection(remote, timeout)
	}

	return remote, []Connection{local}
}

func TestMessageChannel(t *testing.T) {
	testMessageChannel(t, "simple message channel", makeMessageChannel)
}

func TestInProcChannelBehaviorLocal(t *testing.T) {
	testMessageChannel(t, "in-proc, local", makeInProcChannelLocal)
}

func TestInProcChannelBehaviorRemote(t *testing.T) {
	testMessageChannel(t, "in-proc, remote", makeInProcChannelRemote)
}

func BenchmarkMessageChannel(b *testing.B) {
	benchmarkMessageChannel(b, "simple message channel", makeMessageChannel)
}

func BenchmarkMessageChannelBuffered(b *testing.B) {
	benchmarkMessageChannelBuffered(b, "simple message channel", makeMessageChannel)
}

func BenchmarkMessageChannelTimeout(b *testing.B) {
	benchmarkMessageChannelTimeout(b, "simple message channel", makeMessageChannel)
}

func BenchmarkMessageChannelBufferedTimeout(b *testing.B) {
	benchmarkMessageChannelBufferedTimeout(b, "simple message channel", makeMessageChannel)
}

func BenchmarkInProcChannelBehaviorLocal(b *testing.B) {
	benchmarkMessageChannel(b, "in-proc, local", makeInProcChannelLocal)
}

func BenchmarkInProcChannelBehaviorLocalBuffered(b *testing.B) {
	benchmarkMessageChannelBuffered(b, "in-proc, local", makeInProcChannelLocal)
}

func BenchmarkInProcChannelBehaviorLocalTimeout(b *testing.B) {
	benchmarkMessageChannelTimeout(b, "in-proc, local", makeInProcChannelLocal)
}

func BenchmarkInProcChannelBehaviorLocalBufferedTimeout(b *testing.B) {
	benchmarkMessageChannelBufferedTimeout(b, "in-proc, local", makeInProcChannelLocal)
}

func BenchmarkInProcChannelBehaviorRemote(b *testing.B) {
	benchmarkMessageChannel(b, "in-proc, remote", makeInProcChannelRemote)
}

func BenchmarkInProcChannelBehaviorRemoteBuffered(b *testing.B) {
	benchmarkMessageChannelBuffered(b, "in-proc, remote", makeInProcChannelRemote)
}

func BenchmarkInProcChannelBehaviorRemoteTimeout(b *testing.B) {
	benchmarkMessageChannelTimeout(b, "in-proc, remote", makeInProcChannelRemote)
}

func BenchmarkInProcChannelBehaviorRemoteBufferedTimeout(b *testing.B) {
	benchmarkMessageChannelBufferedTimeout(b, "in-proc, remote", makeInProcChannelRemote)
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
