package cast

import (
	"testing"
	"time"
)

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
