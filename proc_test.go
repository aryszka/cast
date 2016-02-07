package cast

import (
	"testing"
	"time"
)

func TestMessageChannel(t *testing.T) {
	testMessageChannel(t, "simple message channel", func(buffer int) (Connection, []Connection) {
		mc := make(MessageChannel, buffer)
		return mc, []Connection{mc}
	})
}

func TestInProcChannelBehaviorLocal(t *testing.T) {
	testMessageChannel(t, "in-proc, local", func(buffer int) (Connection, []Connection) {
		local, remote := NewInProcConnection()
		if buffer != 0 {
			local = NewBufferedConnection(local, buffer)
		}

		return local, []Connection{remote}
	})
}

func TestInProcChannelBehaviorRemote(t *testing.T) {
	testMessageChannel(t, "in-proc, remote", func(buffer int) (Connection, []Connection) {
		local, remote := NewInProcConnection()
		if buffer != 0 {
			remote = NewBufferedConnection(remote, buffer)
		}

		return remote, []Connection{local}
	})
}

func TestTimeoutChannelBehavior(t *testing.T) {
	testMessageChannel(t, "timeout connection", func(buffer int) (Connection, []Connection) {
		mc := make(MessageChannel, buffer)

		tc := NewTimeoutConnection(mc, time.Second)
		go func() {
			<-tc.Timeout
			t.Error("unexpected timeout")
		}()

		return tc, []Connection{mc}
	})
}

func TestBufferChannelBehavior(t *testing.T) {
	testMessageChannel(t, "buffer connection", func(buffer int) (Connection, []Connection) {
		mc := make(MessageChannel)
		bc := NewBufferedConnection(mc, buffer)
		return bc, []Connection{mc}
	})
}

func TestTimeout(t *testing.T) {
	mc := make(MessageChannel)
	tc := NewTimeoutConnection(mc, time.Millisecond)
	m := &Message{}
	go func() { tc.Send() <- m }()
	select {
	case err := <-tc.Timeout:
		if terr, ok := err.(*TimeoutError); ok {
			if terr.Message != m {
				t.Error("invalid message in timeout")
			}
		} else {
			t.Error("invalid error on timeout")
		}
	case <-time.After(2 * time.Millisecond):
		t.Error("failed to timeout")
	}
}
